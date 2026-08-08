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
 * When both host and sandbox directories exist, directory handles are paired
 * and NtQueryDirectoryFile returns a case-insensitive overlay: sandbox names
 * replace host names, while host-only entries remain visible.
 */

#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
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
#define GVHD_STATUS_NO_MORE_FILES           ((NTSTATUS)0x80000006u)
#define GVHD_STATUS_BUFFER_OVERFLOW         ((NTSTATUS)0x80000005u)
#define GVHD_STATUS_BUFFER_TOO_SMALL        ((NTSTATUS)0xC0000023u)

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
typedef NTSTATUS(NTAPI *P_NtQueryDirectoryFile)(
    HANDLE FileHandle, HANDLE Event, PIO_APC_ROUTINE ApcRoutine, PVOID ApcContext,
    PIO_STATUS_BLOCK IoStatusBlock, PVOID FileInformation, ULONG Length,
    FILE_INFORMATION_CLASS FileInformationClass, BOOLEAN ReturnSingleEntry,
    PUNICODE_STRING FileName, BOOLEAN RestartScan);
typedef NTSTATUS(NTAPI *P_NtClose)(HANDLE Handle);

static LPVOID __sys_NtCreateFile = NULL;
static LPVOID __sys_NtOpenFile = NULL;
static LPVOID __sys_NtQueryAttributesFile = NULL;
static LPVOID __sys_NtQueryFullAttributesFile = NULL;
static LPVOID __sys_NtDeleteFile = NULL;
static LPVOID __sys_NtQueryDirectoryFile = NULL;
static LPVOID __sys_NtClose = NULL;

static P_NtCreateFile pfnOrig_NtCreateFile = NULL;
static P_NtOpenFile pfnOrig_NtOpenFile = NULL;
static P_NtQueryAttributesFile pfnOrig_NtQueryAttributesFile = NULL;
static P_NtQueryFullAttributesFile pfnOrig_NtQueryFullAttributesFile = NULL;
static P_NtDeleteFile pfnOrig_NtDeleteFile = NULL;
static P_NtQueryDirectoryFile pfnOrig_NtQueryDirectoryFile = NULL;
static P_NtClose pfnOrig_NtClose = NULL;

/* Helpers called by GetFileAttributesW/CreateDirectoryW must bypass our own
 * hooks, otherwise probing/parent creation would recurse through this file. */
static __thread LONG g_file_reentry;

#define GVHD_DIRECTORY_MAP_MAX 64u
#define GVHD_DIRECTORY_QUERY_MAX 65536u

/* When both the host directory and its sandbox mirror exist, the caller keeps
 * the host handle while the paired sandbox handle is used by the directory
 * query hook.  The map is deliberately bounded: failure to reserve a pair
 * falls back to the normal single-directory behavior. */
struct gvhd_directory_map_entry {
    HANDLE host_handle;
    HANDLE sandbox_handle;
};

static struct gvhd_directory_map_entry g_directory_map[GVHD_DIRECTORY_MAP_MAX];
static CRITICAL_SECTION g_directory_map_lock;
static BOOLEAN g_directory_map_lock_ready;

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
    const struct gvhd_param_block *param = gvhd_get_param();
    const WCHAR *root = param->game_data_root;
    uint32_t encoded = (param->flags & GVHD_PARAM_FLAG_GAME_DRIVE_MASK) >>
                       GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT;

    if (encoded >= 1u && encoded <= 26u) {
        return (WCHAR)(L'a' + encoded - 1u);
    }

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

static BOOLEAN gvhd_directory_map_add(HANDLE host_handle, HANDLE sandbox_handle)
{
    BOOLEAN added = FALSE;

    if (!g_directory_map_lock_ready || host_handle == NULL || sandbox_handle == NULL) {
        return FALSE;
    }
    EnterCriticalSection(&g_directory_map_lock);
    for (size_t i = 0; i < GVHD_DIRECTORY_MAP_MAX; ++i) {
        if (g_directory_map[i].host_handle == NULL) {
            g_directory_map[i].host_handle = host_handle;
            g_directory_map[i].sandbox_handle = sandbox_handle;
            added = TRUE;
            break;
        }
    }
    LeaveCriticalSection(&g_directory_map_lock);
    return added;
}

static HANDLE gvhd_directory_map_find(HANDLE host_handle)
{
    HANDLE sandbox_handle = NULL;

    if (!g_directory_map_lock_ready || host_handle == NULL) {
        return NULL;
    }
    EnterCriticalSection(&g_directory_map_lock);
    for (size_t i = 0; i < GVHD_DIRECTORY_MAP_MAX; ++i) {
        if (g_directory_map[i].host_handle == host_handle) {
            sandbox_handle = g_directory_map[i].sandbox_handle;
            break;
        }
    }
    LeaveCriticalSection(&g_directory_map_lock);
    return sandbox_handle;
}

static HANDLE gvhd_directory_map_remove(HANDLE host_handle)
{
    HANDLE sandbox_handle = NULL;

    if (!g_directory_map_lock_ready || host_handle == NULL) {
        return NULL;
    }
    EnterCriticalSection(&g_directory_map_lock);
    for (size_t i = 0; i < GVHD_DIRECTORY_MAP_MAX; ++i) {
        if (g_directory_map[i].host_handle == host_handle) {
            sandbox_handle = g_directory_map[i].sandbox_handle;
            g_directory_map[i].host_handle = NULL;
            g_directory_map[i].sandbox_handle = NULL;
            break;
        }
    }
    LeaveCriticalSection(&g_directory_map_lock);
    return sandbox_handle;
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

static void gvhd_file_log_directory_merge(const struct gvhd_file_rewritten *rw)
{
    gvhd_log_write(L"FILE_DIRECTORY_MERGE src=%ls dst=%ls",
                   rw->source_dos, rw->target_dos);
}

struct gvhd_directory_record {
    const unsigned char *data;
    ULONG span;
    ULONG name_offset;
    ULONG name_length;
};

static BOOLEAN gvhd_directory_layout(FILE_INFORMATION_CLASS info_class,
                                     ULONG *name_offset, ULONG *length_offset)
{
    switch (info_class) {
    case FileDirectoryInformation:
        *name_offset = (ULONG)offsetof(FILE_DIRECTORY_INFORMATION, FileName);
        *length_offset = (ULONG)offsetof(FILE_DIRECTORY_INFORMATION, FileNameLength);
        return TRUE;
    case FileFullDirectoryInformation:
        *name_offset = (ULONG)offsetof(FILE_FULL_DIR_INFORMATION, FileName);
        *length_offset = (ULONG)offsetof(FILE_FULL_DIR_INFORMATION, FileNameLength);
        return TRUE;
    case FileBothDirectoryInformation:
        *name_offset = (ULONG)offsetof(FILE_BOTH_DIR_INFORMATION, FileName);
        *length_offset = (ULONG)offsetof(FILE_BOTH_DIR_INFORMATION, FileNameLength);
        return TRUE;
    case FileIdFullDirectoryInformation:
        *name_offset = (ULONG)offsetof(FILE_ID_FULL_DIR_INFORMATION, FileName);
        *length_offset = (ULONG)offsetof(FILE_ID_FULL_DIR_INFORMATION, FileNameLength);
        return TRUE;
    case FileIdBothDirectoryInformation:
        *name_offset = (ULONG)offsetof(FILE_ID_BOTH_DIR_INFORMATION, FileName);
        *length_offset = (ULONG)offsetof(FILE_ID_BOTH_DIR_INFORMATION, FileNameLength);
        return TRUE;
    case FileNamesInformation:
        *name_offset = (ULONG)offsetof(FILE_NAMES_INFORMATION, FileName);
        *length_offset = (ULONG)offsetof(FILE_NAMES_INFORMATION, FileNameLength);
        return TRUE;
    default:
        return FALSE;
    }
}

static BOOLEAN gvhd_directory_record_info(const unsigned char *data,
                                          ULONG available,
                                          FILE_INFORMATION_CLASS info_class,
                                          struct gvhd_directory_record *record)
{
    ULONG name_offset;
    ULONG length_offset;
    ULONG next_offset;
    ULONG name_length;
    ULONG minimum_span;
    ULONG name_end;

    if (!gvhd_directory_layout(info_class, &name_offset, &length_offset) ||
        available < length_offset + sizeof(ULONG)) {
        return FALSE;
    }
    name_length = *(const ULONG *)(data + length_offset);
    if ((name_length & 1u) != 0 ||
        name_offset > available ||
        name_length > available - name_offset) {
        return FALSE;
    }
    name_end = name_offset + name_length;
    next_offset = *(const ULONG *)data;
    /* Windows directory records are DWORD-aligned between entries.  The
     * final Wine record may end exactly at FileNameLength, so do not require
     * padding after a record whose NextEntryOffset is zero. */
    minimum_span = next_offset == 0 ? name_end : (name_end + 3u) & ~3u;
    if (minimum_span == 0 || minimum_span > available) {
        return FALSE;
    }
    if (next_offset != 0 &&
        (next_offset < minimum_span || next_offset > available)) {
        return FALSE;
    }

    record->data = data;
    record->span = next_offset == 0 ? minimum_span : next_offset;
    record->name_offset = name_offset;
    record->name_length = name_length;
    return TRUE;
}

static BOOLEAN gvhd_directory_name_equal(
    const struct gvhd_directory_record *left,
    const struct gvhd_directory_record *right)
{
    ULONG left_chars = left->name_length / sizeof(WCHAR);
    ULONG right_chars = right->name_length / sizeof(WCHAR);

    if (left_chars != right_chars) {
        return FALSE;
    }
    for (ULONG i = 0; i < left_chars; ++i) {
        if (!gvhd_file_ieq_char(
                ((const WCHAR *)(left->data + left->name_offset))[i],
                ((const WCHAR *)(right->data + right->name_offset))[i])) {
            return FALSE;
        }
    }
    return TRUE;
}

static BOOLEAN gvhd_directory_collect(
    const unsigned char *buffer, ULONG bytes, FILE_INFORMATION_CLASS info_class,
    struct gvhd_directory_record *records, ULONG *record_count,
    ULONG record_capacity, BOOLEAN overlay)
{
    ULONG offset = 0;

    while (offset < bytes) {
        struct gvhd_directory_record current;
        if (!gvhd_directory_record_info(buffer + offset, bytes - offset,
                                         info_class, &current)) {
            return FALSE;
        }
        if (overlay) {
            BOOLEAN replaced = FALSE;
            for (ULONG i = 0; i < *record_count; ++i) {
                if (gvhd_directory_name_equal(&records[i], &current)) {
                    records[i] = current;
                    replaced = TRUE;
                    break;
                }
            }
            if (!replaced) {
                if (*record_count >= record_capacity) {
                    return FALSE;
                }
                records[(*record_count)++] = current;
            }
        } else {
            if (*record_count >= record_capacity) {
                return FALSE;
            }
            records[(*record_count)++] = current;
        }
        if (*(const ULONG *)(buffer + offset) == 0) {
            break;
        }
        offset += *(const ULONG *)(buffer + offset);
    }
    return TRUE;
}

static NTSTATUS gvhd_directory_emit(
    PVOID output, ULONG output_length,
    const struct gvhd_directory_record *records, ULONG record_count,
    PULONG_PTR information)
{
    unsigned char *destination = (unsigned char *)output;
    ULONG written = 0;
    ULONG previous = 0;
    BOOLEAN have_previous = FALSE;

    for (ULONG i = 0; i < record_count; ++i) {
        const struct gvhd_directory_record *record = &records[i];
        if (record->span > output_length - written) {
            if (!have_previous) {
                *information = 0;
                return GVHD_STATUS_BUFFER_TOO_SMALL;
            }
            break;
        }
        memcpy(destination + written, record->data, record->span);
        *(ULONG *)(destination + written) = 0;
        if (have_previous) {
            *(ULONG *)(destination + previous) = written - previous;
        }
        previous = written;
        written += record->span;
        have_previous = TRUE;
    }
    if (!have_previous) {
        *information = 0;
        return GVHD_STATUS_NO_MORE_FILES;
    }
    *information = written;
    return GVHD_STATUS_SUCCESS;
}

static ULONG_PTR gvhd_directory_status_bytes(const IO_STATUS_BLOCK *io,
                                             NTSTATUS status)
{
    if (io == NULL || status == GVHD_STATUS_NO_MORE_FILES) {
        return 0;
    }
    return io->Information > GVHD_DIRECTORY_QUERY_MAX
        ? GVHD_DIRECTORY_QUERY_MAX
        : io->Information;
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
    NTSTATUS sandbox_status;
    HANDLE sandbox_handle = NULL;
    IO_STATUS_BLOCK sandbox_io;

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

    /* Keep the host directory handle when both views exist.  A paired
     * sandbox handle is consumed by NtQueryDirectoryFile so enumeration can
     * expose host-only entries while still overlaying sandbox names. */
    if (!write_intent && (CreateOptions & FILE_DIRECTORY_FILE) != 0 &&
        gvhd_file_sandbox_exists(rw.target_dos)) {
        status = pfnOrig_NtCreateFile(FileHandle, DesiredAccess, ObjectAttributes,
                                      IoStatusBlock, AllocationSize, FileAttributes,
                                      ShareAccess, CreateDisposition, CreateOptions,
                                      EaBuffer, EaLength);
        if (!NT_SUCCESS(status)) {
            /* The host directory may not exist even though the sandbox view
             * does; the normal sandbox-only path below handles that case. */
        } else {
            memset(&sandbox_io, 0, sizeof(sandbox_io));
            sandbox_status = pfnOrig_NtCreateFile(
                &sandbox_handle, DesiredAccess, &rw.oa, &sandbox_io,
                AllocationSize, FileAttributes, ShareAccess, CreateDisposition,
                CreateOptions, EaBuffer, EaLength);
            if (NT_SUCCESS(sandbox_status) &&
                gvhd_directory_map_add(*FileHandle, sandbox_handle)) {
                gvhd_file_log_directory_merge(&rw);
                return status;
            }
            if (sandbox_handle != NULL) {
                CloseHandle(sandbox_handle);
            }
            return status;
        }
    }

    use_sandbox = write_intent || gvhd_file_sandbox_exists(rw.target_dos);
    if (!use_sandbox) {
        if (!write_intent) {
            gvhd_file_log_readthrough(&rw);
        }
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
    NTSTATUS sandbox_status;
    HANDLE sandbox_handle = NULL;
    IO_STATUS_BLOCK sandbox_io;

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

    if (!write_intent && (OpenOptions & FILE_DIRECTORY_FILE) != 0 &&
        gvhd_file_sandbox_exists(rw.target_dos)) {
        status = pfnOrig_NtOpenFile(FileHandle, DesiredAccess, ObjectAttributes,
                                    IoStatusBlock, ShareAccess, OpenOptions);
        if (!NT_SUCCESS(status)) {
            /* Fall through to the sandbox-only path when the host directory
             * is absent. */
        } else {
            memset(&sandbox_io, 0, sizeof(sandbox_io));
            sandbox_status = pfnOrig_NtOpenFile(
                &sandbox_handle, DesiredAccess, &rw.oa, &sandbox_io,
                ShareAccess, OpenOptions);
            if (NT_SUCCESS(sandbox_status) &&
                gvhd_directory_map_add(*FileHandle, sandbox_handle)) {
                gvhd_file_log_directory_merge(&rw);
                return status;
            }
            if (sandbox_handle != NULL) {
                CloseHandle(sandbox_handle);
            }
            return status;
        }
    }

    use_sandbox = write_intent || gvhd_file_sandbox_exists(rw.target_dos);
    if (!use_sandbox) {
        if (!write_intent) {
            gvhd_file_log_readthrough(&rw);
        }
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

static BOOLEAN gvhd_directory_query_status_supported(NTSTATUS status)
{
    return status == GVHD_STATUS_SUCCESS ||
           status == GVHD_STATUS_NO_MORE_FILES ||
           status == GVHD_STATUS_BUFFER_OVERFLOW;
}

static NTSTATUS NTAPI Hook_NtQueryDirectoryFile(
    HANDLE FileHandle, HANDLE Event, PIO_APC_ROUTINE ApcRoutine, PVOID ApcContext,
    PIO_STATUS_BLOCK IoStatusBlock, PVOID FileInformation, ULONG Length,
    FILE_INFORMATION_CLASS FileInformationClass, BOOLEAN ReturnSingleEntry,
    PUNICODE_STRING FileName, BOOLEAN RestartScan)
{
    HANDLE sandbox_handle;
    unsigned char *host_buffer;
    unsigned char *sandbox_buffer;
    struct gvhd_directory_record *records;
    IO_STATUS_BLOCK host_io;
    IO_STATUS_BLOCK sandbox_io;
    NTSTATUS host_status;
    NTSTATUS sandbox_status;
    ULONG host_bytes;
    ULONG sandbox_bytes;
    ULONG name_offset;
    ULONG length_offset;
    ULONG record_count = 0;
    ULONG_PTR information = 0;
    NTSTATUS result;

    if (g_file_reentry > 0 || pfnOrig_NtQueryDirectoryFile == NULL ||
        (sandbox_handle = gvhd_directory_map_find(FileHandle)) == NULL ||
        Event != NULL || ApcRoutine != NULL || ApcContext != NULL ||
        ReturnSingleEntry ||
        !gvhd_directory_layout(FileInformationClass, &name_offset, &length_offset) ||
        FileInformation == NULL || Length == 0) {
        return pfnOrig_NtQueryDirectoryFile(
            FileHandle, Event, ApcRoutine, ApcContext, IoStatusBlock,
            FileInformation, Length, FileInformationClass, ReturnSingleEntry,
            FileName, RestartScan);
    }

    host_buffer = (unsigned char *)malloc(GVHD_DIRECTORY_QUERY_MAX);
    sandbox_buffer = (unsigned char *)malloc(GVHD_DIRECTORY_QUERY_MAX);
    records = (struct gvhd_directory_record *)calloc(
        GVHD_DIRECTORY_QUERY_MAX / 16u, sizeof(*records));
    if (host_buffer == NULL || sandbox_buffer == NULL || records == NULL) {
        free(host_buffer);
        free(sandbox_buffer);
        free(records);
        return pfnOrig_NtQueryDirectoryFile(
            FileHandle, Event, ApcRoutine, ApcContext, IoStatusBlock,
            FileInformation, Length, FileInformationClass, ReturnSingleEntry,
            FileName, RestartScan);
    }

    memset(&host_io, 0, sizeof(host_io));
    memset(&sandbox_io, 0, sizeof(sandbox_io));
    host_status = pfnOrig_NtQueryDirectoryFile(
        FileHandle, NULL, NULL, NULL, &host_io, host_buffer,
        GVHD_DIRECTORY_QUERY_MAX, FileInformationClass, FALSE,
        FileName, RestartScan);
    sandbox_status = pfnOrig_NtQueryDirectoryFile(
        sandbox_handle, NULL, NULL, NULL, &sandbox_io, sandbox_buffer,
        GVHD_DIRECTORY_QUERY_MAX, FileInformationClass, FALSE,
        FileName, RestartScan);
    host_bytes = (ULONG)gvhd_directory_status_bytes(&host_io, host_status);
    sandbox_bytes = (ULONG)gvhd_directory_status_bytes(&sandbox_io, sandbox_status);

    if (!gvhd_directory_query_status_supported(host_status) ||
        !gvhd_directory_query_status_supported(sandbox_status) ||
        !gvhd_directory_collect(host_buffer, host_bytes, FileInformationClass,
                                records, &record_count,
                                GVHD_DIRECTORY_QUERY_MAX / 16u, FALSE) ||
        !gvhd_directory_collect(sandbox_buffer, sandbox_bytes, FileInformationClass,
                                records, &record_count,
                                GVHD_DIRECTORY_QUERY_MAX / 16u, TRUE)) {
        if (IoStatusBlock != NULL) {
            IoStatusBlock->Information = host_bytes;
        }
        if (host_bytes <= Length) {
            memcpy(FileInformation, host_buffer, host_bytes);
        }
        free(host_buffer);
        free(sandbox_buffer);
        free(records);
        return host_status;
    }

    result = gvhd_directory_emit(
        FileInformation, Length, records, record_count, &information);
    if (IoStatusBlock != NULL) {
        IoStatusBlock->Information = information;
    }

    if (host_status == GVHD_STATUS_NO_MORE_FILES &&
        sandbox_status == GVHD_STATUS_NO_MORE_FILES) {
        HANDLE stale_sandbox = gvhd_directory_map_remove(FileHandle);
        if (stale_sandbox != NULL) {
            CloseHandle(stale_sandbox);
        }
    }

    free(host_buffer);
    free(sandbox_buffer);
    free(records);
    return result;
}

/* A caller may close a directory before consuming the whole enumeration.  A
 * stale host-handle entry is unsafe because Windows can reuse that numeric
 * handle for an unrelated object, so pair cleanup belongs on NtClose as well
 * as the normal end-of-enumeration path. */
static NTSTATUS NTAPI Hook_NtClose(HANDLE Handle)
{
    HANDLE sandbox_handle;

    if (pfnOrig_NtClose == NULL) {
        return GVHD_STATUS_SUCCESS;
    }
    sandbox_handle = gvhd_directory_map_remove(Handle);
    if (sandbox_handle != NULL) {
        pfnOrig_NtClose(sandbox_handle);
    }
    return pfnOrig_NtClose(Handle);
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
    if (!rw.active) {
        return pfnOrig_NtQueryAttributesFile(ObjectAttributes, FileInformation);
    }
    if (!gvhd_file_sandbox_exists(rw.target_dos)) {
        gvhd_file_log_readthrough(&rw);
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
    if (!rw.active) {
        return pfnOrig_NtQueryFullAttributesFile(ObjectAttributes, FileInformation);
    }
    if (!gvhd_file_sandbox_exists(rw.target_dos)) {
        gvhd_file_log_readthrough(&rw);
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
    GVHD_FILE_RESOLVE_OPTIONAL(NtQueryDirectoryFile);
    GVHD_FILE_RESOLVE_OPTIONAL(NtClose);

    if (!g_directory_map_lock_ready) {
        InitializeCriticalSection(&g_directory_map_lock);
        g_directory_map_lock_ready = TRUE;
    }

    GVHD_FILE_CREATE_REQUIRED(NtCreateFile);
    GVHD_FILE_CREATE_REQUIRED(NtOpenFile);
    GVHD_FILE_CREATE_REQUIRED(NtQueryAttributesFile);
    GVHD_FILE_CREATE_OPTIONAL(NtQueryFullAttributesFile);
    GVHD_FILE_CREATE_OPTIONAL(NtDeleteFile);
    GVHD_FILE_CREATE_OPTIONAL(NtQueryDirectoryFile);
    GVHD_FILE_CREATE_OPTIONAL(NtClose);

    GVHD_FILE_ENABLE_REQUIRED(NtCreateFile);
    GVHD_FILE_ENABLE_REQUIRED(NtOpenFile);
    GVHD_FILE_ENABLE_REQUIRED(NtQueryAttributesFile);
    GVHD_FILE_ENABLE_OPTIONAL(NtQueryFullAttributesFile);
    GVHD_FILE_ENABLE_OPTIONAL(NtDeleteFile);
    GVHD_FILE_ENABLE_OPTIONAL(NtQueryDirectoryFile);
    GVHD_FILE_ENABLE_OPTIONAL(NtClose);
    return 0;
}
