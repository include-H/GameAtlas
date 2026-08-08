/*
 * gvhook - file virtualization.
 *
 * File paths are evaluated at the NT boundary, before the game obtains a
 * file handle.  The policy is deliberately conservative around the OS:
 * the game's VHD drive is left alone, C:\Windows and C:\Program Files* are
 * left alone, and every other local drive is copy-on-write virtualized below
 * the configured GameData root.
 *
 * Examples with a game on G: and a user profile C:\Users\Hao:
 *
 *   C:\Users\Hao\Documents\x.ini
 *       -> G:\GameData\Users\Hao\Documents\x.ini
 *   D:\SomeApp\state.dat
 *       -> G:\GameData\HostDrives\D\SomeApp\state.dat
 *   C:\Windows\System32\kernel32.dll
 *       -> passthrough
 *
 * Reads prefer the sandbox copy and fall back to the original path.  Opens
 * with write/create/delete intent always use the sandbox and never fall back
 * to the host on failure.  Parent directories are created under the sandbox
 * with a thread-local re-entry guard.
 *
 * This is the first usable file-redirect implementation.  Directory listing
 * is not yet merged: when both host and sandbox directories exist, enumeration
 * currently observes the selected directory (sandbox first).
 */

#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <wchar.h>

#include <windows.h>
#include <winternl.h>

#include "hook_common.h"
#include "internal.h"
#include "MinHook.h"

#define GVHD_FILE_PATH_MAX 2048u

#define GVHD_STATUS_SUCCESS                 ((NTSTATUS)0)
#define GVHD_STATUS_OBJECT_NAME_NOT_FOUND   ((NTSTATUS)0xC0000034u)
#define GVHD_STATUS_OBJECT_PATH_NOT_FOUND   ((NTSTATUS)0xC000003Au)
#define GVHD_STATUS_ACCESS_DENIED           ((NTSTATUS)0xC0000022u)

typedef NTSTATUS(NTAPI *P_NtCreateFile)(
    PHANDLE FileHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    PIO_STATUS_BLOCK IoStatusBlock, PLARGE_INTEGER AllocationSize, ULONG FileAttributes,
    ULONG ShareAccess, ULONG CreateDisposition, ULONG CreateOptions, PVOID EaBuffer,
    ULONG EaLength);
typedef NTSTATUS(NTAPI *P_NtOpenFile)(
    PHANDLE FileHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    PIO_STATUS_BLOCK IoStatusBlock, ULONG ShareAccess, ULONG OpenOptions);
typedef NTSTATUS(NTAPI *P_NtQueryAttributesFile)(
    POBJECT_ATTRIBUTES ObjectAttributes, PFILE_BASIC_INFORMATION FileInformation);
typedef NTSTATUS(NTAPI *P_NtQueryFullAttributesFile)(
    POBJECT_ATTRIBUTES ObjectAttributes, PFILE_NETWORK_OPEN_INFORMATION FileInformation);
typedef NTSTATUS(NTAPI *P_NtDeleteFile)(POBJECT_ATTRIBUTES ObjectAttributes);

static LPVOID __sys_NtCreateFile = NULL;
static LPVOID __sys_NtOpenFile = NULL;
static LPVOID __sys_NtQueryAttributesFile = NULL;
static LPVOID __sys_NtQueryFullAttributesFile = NULL;
static LPVOID __sys_NtDeleteFile = NULL;

static P_NtCreateFile pfnOrig_NtCreateFile = NULL;
static P_NtOpenFile pfnOrig_NtOpenFile = NULL;
static P_NtQueryAttributesFile pfnOrig_NtQueryAttributesFile = NULL;
static P_NtQueryFullAttributesFile pfnOrig_NtQueryFullAttributesFile = NULL;
static P_NtDeleteFile pfnOrig_NtDeleteFile = NULL;

/* Helpers called by GetFileAttributesW/CreateDirectoryW must bypass our own
 * hooks, otherwise probing/parent creation would recurse through this file. */
static __thread LONG g_file_reentry;

static BOOLEAN gvhd_file_ieq_char(WCHAR a, WCHAR b)
{
    if (a >= L'A' && a <= L'Z') {
        a = (WCHAR)(a + (L'a' - L'A'));
    }
    if (b >= L'A' && b <= L'Z') {
        b = (WCHAR)(b + (L'a' - L'A'));
    }
    return a == b;
}

/* Match a directory itself or a child below it, returning the child suffix. */
static BOOLEAN gvhd_file_under(const WCHAR *path, const WCHAR *base,
                               const WCHAR **rest_out)
{
    size_t base_len = wcslen(base);

    while (base_len > 0 && base[base_len - 1] == L'\\') {
        --base_len;
    }
    for (size_t i = 0; i < base_len; ++i) {
        if (path[i] == L'\0' || !gvhd_file_ieq_char(path[i], base[i])) {
            return FALSE;
        }
    }
    if (path[base_len] == L'\0') {
        *rest_out = path + base_len;
        return TRUE;
    }
    if (path[base_len] != L'\\') {
        return FALSE;
    }
    *rest_out = path + base_len + 1;
    return TRUE;
}

static BOOLEAN gvhd_file_append(WCHAR *out, size_t cap, size_t *len,
                                const WCHAR *text)
{
    size_t n = wcslen(text);

    if (*len + n >= cap) {
        return FALSE;
    }
    memcpy(out + *len, text, n * sizeof(WCHAR));
    *len += n;
    out[*len] = L'\0';
    return TRUE;
}

static BOOLEAN gvhd_file_append_char(WCHAR *out, size_t cap, size_t *len,
                                     WCHAR c)
{
    if (*len + 1 >= cap) {
        return FALSE;
    }
    out[(*len)++] = c;
    out[*len] = L'\0';
    return TRUE;
}

static BOOLEAN gvhd_file_copy_unicode(WCHAR *out, size_t cap,
                                       const UNICODE_STRING *name)
{
    size_t chars;

    if (name == NULL || name->Buffer == NULL) {
        return FALSE;
    }
    chars = (size_t)(name->Length / sizeof(WCHAR));
    if (chars + 1 > cap) {
        return FALSE;
    }
    memcpy(out, name->Buffer, chars * sizeof(WCHAR));
    out[chars] = L'\0';
    return TRUE;
}

/* Return the length of a native DOS-device prefix, retaining it in rewrites. */
static size_t gvhd_file_native_prefix_len(const WCHAR *path)
{
    if (wcsncmp(path, L"\\??\\", 4) == 0 ||
        wcsncmp(path, L"\\\\?\\", 4) == 0) {
        return 4;
    }
    if (wcsncmp(path, L"\\DosDevices\\", 12) == 0) {
        return 12;
    }
    return 0;
}

static BOOLEAN gvhd_file_system_passthrough(const WCHAR *path)
{
    static const WCHAR *const prefixes[] = {
        L"C:\\Windows",
        L"C:\\Program Files",
        L"C:\\Program Files (x86)",
    };

    for (size_t i = 0; i < sizeof(prefixes) / sizeof(prefixes[0]); ++i) {
        const WCHAR *rest;
        if (gvhd_file_under(path, prefixes[i], &rest)) {
            return TRUE;
        }
    }
    return FALSE;
}

static WCHAR gvhd_file_game_drive(void)
{
    const WCHAR *root = gvhd_get_param()->game_data_root;

    if (root[0] != L'\0' && root[1] == L':') {
        WCHAR drive = root[0];
        if (drive >= L'A' && drive <= L'Z') {
            drive = (WCHAR)(drive + (L'a' - L'A'));
        }
        return drive;
    }
    return L'\0';
}

static BOOLEAN gvhd_file_is_drive_path(const WCHAR *path)
{
    return path[0] != L'\0' && path[1] == L':' && path[2] == L'\\';
}

/* Build a DOS path under GameData. Return FALSE for passthrough paths. */
static BOOLEAN gvhd_file_build_target(const WCHAR *source, WCHAR *target,
                                      size_t target_cap)
{
    const struct gvhd_param_block *param = gvhd_get_param();
    const WCHAR *rest;
    const WCHAR *profile = param->user_profile;
    size_t root_len;
    size_t len = 0;
    WCHAR game_drive;

    if (!gvhd_file_is_drive_path(source) || param->game_data_root[0] == L'\0') {
        return FALSE;
    }
    game_drive = gvhd_file_game_drive();
    if (game_drive != L'\0' && gvhd_file_ieq_char(source[0], game_drive)) {
        return FALSE; /* the game/VHD drive is already inside the sandbox */
    }
    /* An external GameData root (for example D:\GameAtlas\Horizon) is the
     * destination, not another host path to virtualize.  This also keeps a
     * game that explicitly opens its own state directory from recursively
     * mapping it below HostDrives\D. */
    if (gvhd_file_under(source, param->game_data_root, &rest)) {
        return FALSE;
    }
    if (gvhd_file_ieq_char(source[0], L'c') &&
        gvhd_file_system_passthrough(source)) {
        return FALSE; /* never virtualize OS/runtime binaries */
    }

    root_len = wcslen(param->game_data_root);
    while (root_len > 0 && param->game_data_root[root_len - 1] == L'\\') {
        --root_len;
    }
    if (root_len == 0 || root_len + 1 >= target_cap) {
        return FALSE;
    }
    memcpy(target, param->game_data_root, root_len * sizeof(WCHAR));
    len = root_len;
    target[len] = L'\0';

    if (profile[0] != L'\0' && gvhd_file_under(source, profile, &rest)) {
        const WCHAR *end = profile + wcslen(profile);
        const WCHAR *last;
        while (end > profile && end[-1] == L'\\') {
            --end;
        }
        last = end;
        while (last > profile && last[-1] != L'\\') {
            --last;
        }
        if (!gvhd_file_append(target, target_cap, &len, L"\\Users\\") ||
            !gvhd_file_append(target, target_cap, &len, last)) {
            return FALSE;
        }
        if (*rest != L'\0' &&
            (!gvhd_file_append_char(target, target_cap, &len, L'\\') ||
             !gvhd_file_append(target, target_cap, &len, rest))) {
            return FALSE;
        }
        return TRUE;
    }

    /* Other local drives are kept separate to avoid collisions. */
    if (!gvhd_file_append(target, target_cap, &len, L"\\HostDrives\\") ||
        !gvhd_file_append_char(target, target_cap, &len, source[0]) ||
        (source[3] != L'\0' &&
         (!gvhd_file_append_char(target, target_cap, &len, L'\\') ||
          !gvhd_file_append(target, target_cap, &len, source + 3)))) {
        return FALSE;
    }
    return TRUE;
}

struct gvhd_file_rewritten {
    BOOLEAN active;
    OBJECT_ATTRIBUTES oa;
    UNICODE_STRING name;
    WCHAR source_native[GVHD_FILE_PATH_MAX];
    WCHAR source_dos[GVHD_FILE_PATH_MAX];
    WCHAR target_dos[GVHD_FILE_PATH_MAX];
    WCHAR target_native[GVHD_FILE_PATH_MAX];
};

static void gvhd_file_eval_oa(const OBJECT_ATTRIBUTES *input,
                              struct gvhd_file_rewritten *rw)
{
    size_t native_prefix;
    size_t target_len;

    memset(rw, 0, sizeof(*rw));
    if (input == NULL || input->RootDirectory != NULL ||
        input->ObjectName == NULL ||
        !gvhd_file_copy_unicode(rw->source_native, GVHD_FILE_PATH_MAX,
                                input->ObjectName)) {
        return;
    }
    native_prefix = gvhd_file_native_prefix_len(rw->source_native);
    if (!gvhd_file_copy_unicode(rw->source_dos, GVHD_FILE_PATH_MAX,
                                input->ObjectName) ||
        native_prefix >= wcslen(rw->source_dos)) {
        return;
    }
    /* The second copy above is intentionally replaced with the DOS suffix. */
    memmove(rw->source_dos, rw->source_native + native_prefix,
            (wcslen(rw->source_native + native_prefix) + 1) * sizeof(WCHAR));
    if (!gvhd_file_build_target(rw->source_dos, rw->target_dos,
                                GVHD_FILE_PATH_MAX)) {
        return;
    }
    target_len = wcslen(rw->target_dos);
    if (native_prefix + target_len + 1 > GVHD_FILE_PATH_MAX) {
        return;
    }
    memcpy(rw->target_native, rw->source_native, native_prefix * sizeof(WCHAR));
    memcpy(rw->target_native + native_prefix, rw->target_dos,
           (target_len + 1) * sizeof(WCHAR));

    rw->name.Buffer = rw->target_native;
    rw->name.Length = (USHORT)(wcslen(rw->target_native) * sizeof(WCHAR));
    rw->name.MaximumLength = (USHORT)(GVHD_FILE_PATH_MAX * sizeof(WCHAR));
    rw->oa = *input;
    rw->oa.ObjectName = &rw->name;
    rw->active = TRUE;
}

static BOOLEAN gvhd_file_sandbox_exists(const WCHAR *path)
{
    DWORD attributes;

    ++g_file_reentry;
    attributes = GetFileAttributesW(path);
    --g_file_reentry;
    return attributes != INVALID_FILE_ATTRIBUTES;
}

/* Create every missing parent below the target drive root. */
static BOOLEAN gvhd_file_ensure_parent(const WCHAR *path)
{
    WCHAR parent[GVHD_FILE_PATH_MAX];
    WCHAR partial[GVHD_FILE_PATH_MAX];
    WCHAR *last;
    size_t parent_len;

    if (wcslen(path) >= GVHD_FILE_PATH_MAX) {
        return FALSE;
    }
    wcscpy(parent, path);
    last = wcsrchr(parent, L'\\');
    if (last == NULL || last <= parent + 2) {
        return TRUE;
    }
    *last = L'\0';
    parent_len = wcslen(parent);
    if (parent_len >= GVHD_FILE_PATH_MAX) {
        return FALSE;
    }
    wcscpy(partial, parent);

    ++g_file_reentry;
    for (size_t i = 3; i <= parent_len; ++i) {
        if (partial[i] == L'\\' || i == parent_len) {
            WCHAR saved = partial[i];
            DWORD error;
            partial[i] = L'\0';
            if (partial[0] != L'\0' &&
                !CreateDirectoryW(partial, NULL) &&
                (error = GetLastError()) != ERROR_ALREADY_EXISTS) {
                partial[i] = saved;
                --g_file_reentry;
                return FALSE;
            }
            partial[i] = saved;
        }
    }
    --g_file_reentry;
    return TRUE;
}

static BOOLEAN gvhd_file_is_write_access(ACCESS_MASK access)
{
    const ACCESS_MASK writes = GENERIC_WRITE | GENERIC_ALL |
        FILE_WRITE_DATA | FILE_APPEND_DATA | FILE_WRITE_EA |
        FILE_WRITE_ATTRIBUTES | DELETE | WRITE_DAC | WRITE_OWNER;
    return (access & writes) != 0;
}

static BOOLEAN gvhd_file_is_read_fallback_status(NTSTATUS status)
{
    return status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND ||
           status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
           status == GVHD_STATUS_ACCESS_DENIED;
}

static void gvhd_file_log_rewrite(const WCHAR *operation,
                                  const struct gvhd_file_rewritten *rw)
{
    gvhd_log_write(L"FILE_REWRITE op=%ls src=%ls dst=%ls",
                   operation, rw->source_dos, rw->target_dos);
}

static void gvhd_file_log_readthrough(const struct gvhd_file_rewritten *rw)
{
    gvhd_log_write(L"FILE_READTHROUGH src=%ls", rw->source_dos);
}

static NTSTATUS NTAPI Hook_NtCreateFile(
    PHANDLE FileHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    PIO_STATUS_BLOCK IoStatusBlock, PLARGE_INTEGER AllocationSize, ULONG FileAttributes,
    ULONG ShareAccess, ULONG CreateDisposition, ULONG CreateOptions, PVOID EaBuffer,
    ULONG EaLength)
{
    struct gvhd_file_rewritten rw;
    BOOLEAN write_intent;
    BOOLEAN use_sandbox;
    NTSTATUS status;

    if (g_file_reentry > 0 || pfnOrig_NtCreateFile == NULL) {
        return pfnOrig_NtCreateFile(FileHandle, DesiredAccess, ObjectAttributes,
                                    IoStatusBlock, AllocationSize, FileAttributes,
                                    ShareAccess, CreateDisposition, CreateOptions,
                                    EaBuffer, EaLength);
    }
    gvhd_file_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtCreateFile(FileHandle, DesiredAccess, ObjectAttributes,
                                    IoStatusBlock, AllocationSize, FileAttributes,
                                    ShareAccess, CreateDisposition, CreateOptions,
                                    EaBuffer, EaLength);
    }

    write_intent = gvhd_file_is_write_access(DesiredAccess) ||
                   CreateDisposition != FILE_OPEN ||
                   (CreateOptions & FILE_DELETE_ON_CLOSE) != 0;
    use_sandbox = write_intent || gvhd_file_sandbox_exists(rw.target_dos);
    if (!use_sandbox) {
        return pfnOrig_NtCreateFile(FileHandle, DesiredAccess, ObjectAttributes,
                                    IoStatusBlock, AllocationSize, FileAttributes,
                                    ShareAccess, CreateDisposition, CreateOptions,
                                    EaBuffer, EaLength);
    }
    if (write_intent && !gvhd_file_ensure_parent(rw.target_dos)) {
        return GVHD_STATUS_OBJECT_PATH_NOT_FOUND;
    }

    gvhd_file_log_rewrite(L"NtCreateFile", &rw);
    status = pfnOrig_NtCreateFile(FileHandle, DesiredAccess, &rw.oa,
                                   IoStatusBlock, AllocationSize, FileAttributes,
                                   ShareAccess, CreateDisposition, CreateOptions,
                                   EaBuffer, EaLength);
    if (!write_intent && gvhd_file_is_read_fallback_status(status)) {
        gvhd_file_log_readthrough(&rw);
        return pfnOrig_NtCreateFile(FileHandle, DesiredAccess, ObjectAttributes,
                                    IoStatusBlock, AllocationSize, FileAttributes,
                                    ShareAccess, CreateDisposition, CreateOptions,
                                    EaBuffer, EaLength);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtOpenFile(
    PHANDLE FileHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    PIO_STATUS_BLOCK IoStatusBlock, ULONG ShareAccess, ULONG OpenOptions)
{
    struct gvhd_file_rewritten rw;
    BOOLEAN write_intent;
    BOOLEAN use_sandbox;
    NTSTATUS status;

    if (g_file_reentry > 0 || pfnOrig_NtOpenFile == NULL) {
        return pfnOrig_NtOpenFile(FileHandle, DesiredAccess, ObjectAttributes,
                                  IoStatusBlock, ShareAccess, OpenOptions);
    }
    gvhd_file_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtOpenFile(FileHandle, DesiredAccess, ObjectAttributes,
                                  IoStatusBlock, ShareAccess, OpenOptions);
    }
    write_intent = gvhd_file_is_write_access(DesiredAccess);
    use_sandbox = write_intent || gvhd_file_sandbox_exists(rw.target_dos);
    if (!use_sandbox) {
        return pfnOrig_NtOpenFile(FileHandle, DesiredAccess, ObjectAttributes,
                                  IoStatusBlock, ShareAccess, OpenOptions);
    }
    if (write_intent && !gvhd_file_ensure_parent(rw.target_dos)) {
        return GVHD_STATUS_OBJECT_PATH_NOT_FOUND;
    }

    gvhd_file_log_rewrite(L"NtOpenFile", &rw);
    status = pfnOrig_NtOpenFile(FileHandle, DesiredAccess, &rw.oa,
                                IoStatusBlock, ShareAccess, OpenOptions);
    if (!write_intent && gvhd_file_is_read_fallback_status(status)) {
        gvhd_file_log_readthrough(&rw);
        return pfnOrig_NtOpenFile(FileHandle, DesiredAccess, ObjectAttributes,
                                  IoStatusBlock, ShareAccess, OpenOptions);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtQueryAttributesFile(
    POBJECT_ATTRIBUTES ObjectAttributes, PFILE_BASIC_INFORMATION FileInformation)
{
    struct gvhd_file_rewritten rw;
    NTSTATUS status;

    if (g_file_reentry > 0 || pfnOrig_NtQueryAttributesFile == NULL) {
        return pfnOrig_NtQueryAttributesFile(ObjectAttributes, FileInformation);
    }
    gvhd_file_eval_oa(ObjectAttributes, &rw);
    if (!rw.active || !gvhd_file_sandbox_exists(rw.target_dos)) {
        return pfnOrig_NtQueryAttributesFile(ObjectAttributes, FileInformation);
    }
    gvhd_file_log_rewrite(L"NtQueryAttributesFile", &rw);
    status = pfnOrig_NtQueryAttributesFile(&rw.oa, FileInformation);
    if (gvhd_file_is_read_fallback_status(status)) {
        gvhd_file_log_readthrough(&rw);
        return pfnOrig_NtQueryAttributesFile(ObjectAttributes, FileInformation);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtQueryFullAttributesFile(
    POBJECT_ATTRIBUTES ObjectAttributes, PFILE_NETWORK_OPEN_INFORMATION FileInformation)
{
    struct gvhd_file_rewritten rw;
    NTSTATUS status;

    if (g_file_reentry > 0 || pfnOrig_NtQueryFullAttributesFile == NULL) {
        return pfnOrig_NtQueryFullAttributesFile(ObjectAttributes, FileInformation);
    }
    gvhd_file_eval_oa(ObjectAttributes, &rw);
    if (!rw.active || !gvhd_file_sandbox_exists(rw.target_dos)) {
        return pfnOrig_NtQueryFullAttributesFile(ObjectAttributes, FileInformation);
    }
    gvhd_file_log_rewrite(L"NtQueryFullAttributesFile", &rw);
    status = pfnOrig_NtQueryFullAttributesFile(&rw.oa, FileInformation);
    if (gvhd_file_is_read_fallback_status(status)) {
        gvhd_file_log_readthrough(&rw);
        return pfnOrig_NtQueryFullAttributesFile(ObjectAttributes, FileInformation);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtDeleteFile(POBJECT_ATTRIBUTES ObjectAttributes)
{
    struct gvhd_file_rewritten rw;

    if (g_file_reentry > 0 || pfnOrig_NtDeleteFile == NULL) {
        return pfnOrig_NtDeleteFile(ObjectAttributes);
    }
    gvhd_file_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtDeleteFile(ObjectAttributes);
    }
    if (!gvhd_file_sandbox_exists(rw.target_dos)) {
        return GVHD_STATUS_OBJECT_NAME_NOT_FOUND;
    }
    gvhd_file_log_rewrite(L"NtDeleteFile", &rw);
    return pfnOrig_NtDeleteFile(&rw.oa);
}

#define GVHD_FILE_RESOLVE_REQUIRED(FN)                                         \
    do {                                                                        \
        __sys_##FN = (LPVOID)GetProcAddress(hNtdll, #FN);                       \
        if (__sys_##FN == NULL) {                                              \
            gvhd_log_write(L"FILE_HOOK_RESOLVE_FAILED fn=%S", #FN);           \
            return GVHD_INIT_ERR_HOOK;                                         \
        }                                                                       \
    } while (0)

#define GVHD_FILE_RESOLVE_OPTIONAL(FN)                                         \
    do {                                                                        \
        __sys_##FN = (LPVOID)GetProcAddress(hNtdll, #FN);                       \
        if (__sys_##FN == NULL) {                                              \
            gvhd_log_write(L"FILE_HOOK_OPTIONAL_SKIPPED fn=%S", #FN);          \
        }                                                                       \
    } while (0)

#define GVHD_FILE_CREATE_REQUIRED(FN)                                          \
    do {                                                                        \
        mh = MH_CreateHook(__sys_##FN, (LPVOID)&Hook_##FN,                      \
                           (LPVOID *)&pfnOrig_##FN);                            \
        if (mh != MH_OK) {                                                      \
            gvhd_log_write(L"FILE_HOOK_CREATE_FAILED fn=%S mh=%d", #FN,       \
                           (int)mh);                                            \
            return GVHD_INIT_ERR_HOOK;                                          \
        }                                                                       \
    } while (0)

#define GVHD_FILE_CREATE_OPTIONAL(FN)                                          \
    do {                                                                        \
        if (__sys_##FN != NULL) {                                               \
            mh = MH_CreateHook(__sys_##FN, (LPVOID)&Hook_##FN,                  \
                               (LPVOID *)&pfnOrig_##FN);                        \
            if (mh != MH_OK) {                                                  \
                gvhd_log_write(L"FILE_HOOK_OPTIONAL_CREATE_FAILED fn=%S mh=%d",\
                               #FN, (int)mh);                                   \
                __sys_##FN = NULL;                                              \
            }                                                                   \
        }                                                                       \
    } while (0)

#define GVHD_FILE_ENABLE_REQUIRED(FN)                                          \
    do {                                                                        \
        mh = MH_EnableHook(__sys_##FN);                                         \
        if (mh != MH_OK) {                                                      \
            gvhd_log_write(L"FILE_HOOK_ENABLE_FAILED fn=%S mh=%d", #FN,       \
                           (int)mh);                                            \
            return GVHD_INIT_ERR_HOOK;                                          \
        }                                                                       \
    } while (0)

#define GVHD_FILE_ENABLE_OPTIONAL(FN)                                          \
    do {                                                                        \
        if (pfnOrig_##FN != NULL) {                                             \
            mh = MH_EnableHook(__sys_##FN);                                     \
            if (mh != MH_OK) {                                                  \
                gvhd_log_write(L"FILE_HOOK_OPTIONAL_ENABLE_FAILED fn=%S mh=%d",\
                               #FN, (int)mh);                                   \
            }                                                                   \
        }                                                                       \
    } while (0)

uint32_t gvhd_install_file_hooks(void)
{
    HMODULE hNtdll;
    MH_STATUS mh;

    hNtdll = GetModuleHandleW(L"ntdll.dll");
    if (hNtdll == NULL) {
        return GVHD_INIT_ERR_HOOK;
    }

    GVHD_FILE_RESOLVE_REQUIRED(NtCreateFile);
    GVHD_FILE_RESOLVE_REQUIRED(NtOpenFile);
    GVHD_FILE_RESOLVE_REQUIRED(NtQueryAttributesFile);
    GVHD_FILE_RESOLVE_OPTIONAL(NtQueryFullAttributesFile);
    GVHD_FILE_RESOLVE_OPTIONAL(NtDeleteFile);

    GVHD_FILE_CREATE_REQUIRED(NtCreateFile);
    GVHD_FILE_CREATE_REQUIRED(NtOpenFile);
    GVHD_FILE_CREATE_REQUIRED(NtQueryAttributesFile);
    GVHD_FILE_CREATE_OPTIONAL(NtQueryFullAttributesFile);
    GVHD_FILE_CREATE_OPTIONAL(NtDeleteFile);

    GVHD_FILE_ENABLE_REQUIRED(NtCreateFile);
    GVHD_FILE_ENABLE_REQUIRED(NtOpenFile);
    GVHD_FILE_ENABLE_REQUIRED(NtQueryAttributesFile);
    GVHD_FILE_ENABLE_OPTIONAL(NtQueryFullAttributesFile);
    GVHD_FILE_ENABLE_OPTIONAL(NtDeleteFile);
    return 0;
}
