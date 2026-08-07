/*
 * file_mode.c — file-redirect tests (stage-2 acceptance, W3T10).
 *
 * // allow: SIZE_OK — one cohesive module: 8 mechanically-parallel Win32
 * // actions (same marker/error contract, shared helpers, one dispatch)
 * // behind a single mode, per codebase one-mode-per-file convention.
 *
 * One action per invocation (single-action contract):
 *
 *   test-app.exe file new <path>
 *   test-app.exe file --action overwrite <path>
 *   ...
 *
 * argv[0] = mode name ("file"), argv[1..] = mode args. The action is named
 * either as the first positional (short form) or via --action <name>
 * (argv[1] must be "--action" exactly, followed by the name).
 *
 * All paths are Windows paths passed through verbatim (backslashes allowed).
 * They are converted to wide chars (CP_ACP) for the Win32 APIs and echoed
 * back in markers as the original ANSI argv string, so assert scripts see
 * byte-identical paths.
 *
 * The sandbox hook (W3T9, parallel) redirects writes under
 * %USERPROFILE%\Documents / Saved Games / AppData to
 * E:\GameData\Users\<user>\... with read-through; this mode never knows a
 * redirect happened — it just performs the Win32 call on the given path.
 *
 * Output contract (assert_fs.ps1, W3T11, regex-matches these exact markers;
 * every line goes through tlogf() so it lands BOTH on stdout and in the
 * --log file):
 *
 *   [test-app] FILE_MODE_START
 *   [test-app] FILE_NEW <path> OK|EXISTS|ERR=<err>
 *   [test-app] FILE_OVERWRITE <path> OK|ERR=<err>
 *   [test-app] FILE_APPEND <path> OK|ERR=<err>
 *   [test-app] FILE_READ <path> CONTENT=<line>|NOT_FOUND|ERR=<err>
 *   [test-app] FILE_LIST <path> ENTRY=<name>        (one per entry, sorted)
 *   [test-app] FILE_DELETE <path> OK|NOT_FOUND|ERR=<err>
 *   [test-app] FILE_RENAME <old> -> <new> OK|ERR=<err>
 *   [test-app] FILE_EXISTS <path> YES|NO
 *   [test-app] UNKNOWN_ACTION <name>                (exit 2)
 *   [test-app] MISSING_ARG <action>                 (exit 2)
 *
 * Exit code: 0 = every outcome OK/YES; 1 = any outcome was
 * EXISTS/NOT_FOUND/NO/ERR=; 2 = usage error (unknown action, missing arg).
 *
 * Determinism: FILE_LIST entries are sorted (wcscmp) regardless of NTFS
 * enumeration order; FILE_READ CONTENT is the literal first line (written
 * content is ASCII and test-controlled); the only varying token inside
 * markers is the process ID embedded in written content
 * ("GVHD_TEST_<TAG>_<pid>\n").
 *
 * Semantics: new = CREATE_NEW (fails on existing → EXISTS); overwrite =
 * CREATE_ALWAYS; append = OPEN_ALWAYS + FILE_APPEND_DATA (creates when
 * missing, always writes at EOF); read/delete/report NOT_FOUND when the
 * file or its parent directory is missing; rename = MoveFileW (fails with
 * ERR if the destination already exists).
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <windows.h>
#include <wchar.h>

#include "test_app_log.h"

/* Heap copy of an ANSI path converted to UTF-16 (CP_ACP). NULL on failure. */
static wchar_t *to_wide(const char *s)
{
    int n = MultiByteToWideChar(CP_ACP, 0, s, -1, NULL, 0);
    if (n <= 0) {
        return NULL;
    }
    wchar_t *w = malloc((size_t)n * sizeof(wchar_t));
    if (w == NULL) {
        return NULL;
    }
    if (MultiByteToWideChar(CP_ACP, 0, s, -1, w, n) != n) {
        free(w);
        return NULL;
    }
    return w;
}

/* Write "GVHD_TEST_<TAG>_<pid>\n" at the current position. 0 on success. */
static int write_test_line(HANDLE h, const char *tag)
{
    char buf[128];
    int n = snprintf(buf, sizeof(buf), "GVHD_TEST_%s_%lu\n", tag,
                     (unsigned long)GetCurrentProcessId());
    if (n < 0 || (size_t)n >= sizeof(buf)) {
        return -1;
    }
    DWORD written = 0;
    if (!WriteFile(h, buf, (DWORD)n, &written, NULL) ||
        written != (DWORD)n) {
        return -1;
    }
    return 0;
}

/* Create-then-write actions (new/overwrite/append) sharing the marker shape
 * (OK | EXISTS | ERR=). `access`/`disposition` carry the per-action Win32
 * semantics: new = GENERIC_WRITE/CREATE_NEW (EXISTS on existing file),
 * overwrite = GENERIC_WRITE/CREATE_ALWAYS, append = FILE_APPEND_DATA/OPEN_ALWAYS
 * (writes always land at EOF). */
static int act_create(const char *path, const char *tag, const char *marker,
                      DWORD access, DWORD disposition)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        tlogf(stdout, "[test-app] %s %s ERR=oom", marker, path);
        return 1;
    }
    HANDLE h = CreateFileW(wp, access, FILE_SHARE_READ | FILE_SHARE_WRITE,
                           NULL, disposition, FILE_ATTRIBUTE_NORMAL, NULL);
    free(wp);
    if (h == INVALID_HANDLE_VALUE) {
        DWORD gle = GetLastError();
        if (disposition == CREATE_NEW &&
            (gle == ERROR_FILE_EXISTS || gle == ERROR_ALREADY_EXISTS)) {
            tlogf(stdout, "[test-app] %s %s EXISTS", marker, path);
        } else {
            tlogf(stdout, "[test-app] %s %s ERR=%lu", marker, path,
                  (unsigned long)gle);
        }
        return 1;
    }
    int rc;
    if (write_test_line(h, tag) != 0) {
        tlogf(stdout, "[test-app] %s %s ERR=%lu", marker, path,
              (unsigned long)GetLastError());
        rc = 1;
    } else {
        tlogf(stdout, "[test-app] %s %s OK", marker, path);
        rc = 0;
    }
    CloseHandle(h);
    return rc;
}

static int act_new(const char *path)
{
    return act_create(path, "NEW", "FILE_NEW", GENERIC_WRITE, CREATE_NEW);
}

static int act_overwrite(const char *path)
{
    return act_create(path, "OVERWRITE", "FILE_OVERWRITE", GENERIC_WRITE,
                      CREATE_ALWAYS);
}

static int act_append(const char *path)
{
    return act_create(path, "APPEND", "FILE_APPEND", FILE_APPEND_DATA,
                      OPEN_ALWAYS);
}

/* Read the whole file; FILE_READ prints the literal first line. */
static int act_read(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        tlogf(stdout, "[test-app] FILE_READ %s ERR=oom", path);
        return 1;
    }
    HANDLE h = CreateFileW(wp, GENERIC_READ, FILE_SHARE_READ | FILE_SHARE_WRITE,
                           NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
    free(wp);
    if (h == INVALID_HANDLE_VALUE) {
        DWORD gle = GetLastError();
        if (gle == ERROR_FILE_NOT_FOUND || gle == ERROR_PATH_NOT_FOUND) {
            tlogf(stdout, "[test-app] FILE_READ %s NOT_FOUND", path);
        } else {
            tlogf(stdout, "[test-app] FILE_READ %s ERR=%lu", path,
                  (unsigned long)gle);
        }
        return 1;
    }

    size_t cap = 4096;
    size_t len = 0;
    char *buf = malloc(cap);
    if (buf == NULL) {
        tlogf(stdout, "[test-app] FILE_READ %s ERR=oom", path);
        CloseHandle(h);
        return 1;
    }
    int rc = 0;
    for (;;) {
        if (cap - len < 1024) {
            char *nb = realloc(buf, cap * 2);
            if (nb == NULL) {
                tlogf(stdout, "[test-app] FILE_READ %s ERR=oom", path);
                rc = 1;
                break;
            }
            buf = nb;
            cap *= 2;
        }
        DWORD got = 0;
        if (!ReadFile(h, buf + len, (DWORD)(cap - len), &got, NULL)) {
            len += got; /* partial bytes may have been read */
            tlogf(stdout, "[test-app] FILE_READ %s ERR=%lu", path,
                  (unsigned long)GetLastError());
            rc = 1;
            break;
        }
        if (got == 0) {
            break;
        }
        len += got;
    }
    CloseHandle(h);

    if (rc == 0) {
        size_t i = 0;
        while (i < len && buf[i] != '\n') {
            i++;
        }
        if (i > 0 && buf[i - 1] == '\r') {
            i--;
        }
        tlogf(stdout, "[test-app] FILE_READ %s CONTENT=%.*s", path,
              (int)i, buf);
    }
    free(buf);
    return rc;
}

static int wcscmp_qsort(const void *a, const void *b)
{
    const wchar_t *pa = *(const wchar_t *const *)a;
    const wchar_t *pb = *(const wchar_t *const *)b;
    return wcscmp(pa, pb);
}

/* One FILE_LIST ENTRY marker; wide name converted back to ANSI. */
static void print_entry(const char *path, const wchar_t *name)
{
    char buf[1024];
    int n = WideCharToMultiByte(CP_ACP, 0, name, -1, buf,
                                (int)sizeof(buf), NULL, NULL);
    if (n <= 0) {
        tlogf(stdout, "[test-app] FILE_LIST %s ENTRY=<unconvertible>", path);
        return;
    }
    tlogf(stdout, "[test-app] FILE_LIST %s ENTRY=%s", path, buf);
}

static int act_list(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        tlogf(stdout, "[test-app] FILE_LIST %s ERR=oom", path);
        return 1;
    }
    size_t plen = wcslen(wp);
    int has_sep = plen > 0 && (wp[plen - 1] == L'\\' || wp[plen - 1] == L'/');
    size_t pat_len = plen + (has_sep ? 1 : 2) + 1;
    wchar_t *pat = malloc(pat_len * sizeof(wchar_t));
    if (pat == NULL) {
        free(wp);
        tlogf(stdout, "[test-app] FILE_LIST %s ERR=oom", path);
        return 1;
    }
    swprintf(pat, pat_len, has_sep ? L"%ls*" : L"%ls\\*", wp);
    free(wp);

    WIN32_FIND_DATAW fd;
    HANDLE h = FindFirstFileW(pat, &fd);
    free(pat);
    if (h == INVALID_HANDLE_VALUE) {
        DWORD gle = GetLastError();
        if (gle == ERROR_FILE_NOT_FOUND || gle == ERROR_PATH_NOT_FOUND) {
            tlogf(stdout, "[test-app] FILE_LIST %s NOT_FOUND", path);
        } else {
            tlogf(stdout, "[test-app] FILE_LIST %s ERR=%lu", path,
                  (unsigned long)gle);
        }
        return 1;
    }

    wchar_t **names = NULL;
    size_t count = 0;
    size_t cap = 0;
    int rc = 0;
    for (;;) {
        if (wcscmp(fd.cFileName, L".") != 0 && wcscmp(fd.cFileName, L"..") != 0) {
            if (count == cap) {
                size_t ncap = cap == 0 ? 16 : cap * 2;
                wchar_t **nn = realloc(names, ncap * sizeof(*nn));
                if (nn == NULL) {
                    tlogf(stdout, "[test-app] FILE_LIST %s ERR=oom", path);
                    rc = 1;
                    break;
                }
                names = nn;
                cap = ncap;
            }
            names[count] = wcsdup(fd.cFileName);
            if (names[count] == NULL) {
                tlogf(stdout, "[test-app] FILE_LIST %s ERR=oom", path);
                rc = 1;
                break;
            }
            count++;
        }
        if (!FindNextFileW(h, &fd)) {
            break; /* ERROR_NO_MORE_FILES, or mid-enum failure: stop */
        }
    }
    CloseHandle(h);

    if (rc == 0) {
        qsort(names, count, sizeof(*names), wcscmp_qsort);
        size_t i;
        for (i = 0; i < count; i++) {
            print_entry(path, names[i]);
        }
    }
    size_t i;
    for (i = 0; i < count; i++) {
        free(names[i]);
    }
    free(names);
    return rc;
}

static int act_delete(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        tlogf(stdout, "[test-app] FILE_DELETE %s ERR=oom", path);
        return 1;
    }
    int rc;
    if (DeleteFileW(wp)) {
        tlogf(stdout, "[test-app] FILE_DELETE %s OK", path);
        rc = 0;
    } else {
        DWORD gle = GetLastError();
        if (gle == ERROR_FILE_NOT_FOUND || gle == ERROR_PATH_NOT_FOUND) {
            tlogf(stdout, "[test-app] FILE_DELETE %s NOT_FOUND", path);
        } else {
            tlogf(stdout, "[test-app] FILE_DELETE %s ERR=%lu", path,
                  (unsigned long)gle);
        }
        rc = 1;
    }
    free(wp);
    return rc;
}

static int act_rename(const char *old_path, const char *new_path)
{
    wchar_t *wo = to_wide(old_path);
    wchar_t *wn = to_wide(new_path);
    if (wo == NULL || wn == NULL) {
        free(wo);
        free(wn);
        tlogf(stdout, "[test-app] FILE_RENAME %s -> %s ERR=oom", old_path,
              new_path);
        return 1;
    }
    int rc;
    if (MoveFileW(wo, wn)) {
        tlogf(stdout, "[test-app] FILE_RENAME %s -> %s OK", old_path,
              new_path);
        rc = 0;
    } else {
        tlogf(stdout, "[test-app] FILE_RENAME %s -> %s ERR=%lu", old_path,
              new_path, (unsigned long)GetLastError());
        rc = 1;
    }
    free(wo);
    free(wn);
    return rc;
}

static int act_exists(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        tlogf(stdout, "[test-app] FILE_EXISTS %s NO", path);
        return 1;
    }
    DWORD attrs = GetFileAttributesW(wp);
    free(wp);
    if (attrs != INVALID_FILE_ATTRIBUTES) {
        tlogf(stdout, "[test-app] FILE_EXISTS %s YES", path);
        return 0;
    }
    tlogf(stdout, "[test-app] FILE_EXISTS %s NO", path);
    return 1;
}

static void print_action_help(void)
{
    tlogf(stdout, "[test-app] file mode actions (usage: test-app file [--action <name>] <args...>):");
    tlogf(stdout, "  new <path>          create file, fail if exists (EXISTS)");
    tlogf(stdout, "  overwrite <path>    create-or-truncate");
    tlogf(stdout, "  append <path>       append, create when missing");
    tlogf(stdout, "  read <path>         print first line as CONTENT=<line>");
    tlogf(stdout, "  list <path>         enumerate dir, one ENTRY= per sorted name");
    tlogf(stdout, "  delete <path>       delete file");
    tlogf(stdout, "  rename <old> <new>  rename (move) file");
    tlogf(stdout, "  exists <path>       YES|NO attribute check");
    tlogf(stdout, "  exit: 0 all OK, 1 any failure, 2 usage error");
}

int file_mode(int argc, char **argv)
{
    tlogf(stdout, "[test-app] FILE_MODE_START");

    const char *action;
    int args_at;
    int i;

    for (i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--help") == 0) {
            print_action_help();
            return 0;
        }
    }

    if (argc >= 2 && strcmp(argv[1], "--action") == 0) {
        if (argc < 3) {
            tlogf(stdout, "[test-app] MISSING_ARG action");
            return 2;
        }
        action = argv[2];
        args_at = 3;
    } else {
        if (argc < 2) {
            tlogf(stdout, "[test-app] MISSING_ARG action");
            return 2;
        }
        action = argv[1];
        args_at = 2;
    }

    if (strcmp(action, "new") == 0 || strcmp(action, "overwrite") == 0 ||
        strcmp(action, "append") == 0 || strcmp(action, "read") == 0 ||
        strcmp(action, "list") == 0 || strcmp(action, "delete") == 0 ||
        strcmp(action, "exists") == 0) {
        if (args_at >= argc) {
            tlogf(stdout, "[test-app] MISSING_ARG %s", action);
            return 2;
        }
    } else if (strcmp(action, "rename") == 0) {
        if (args_at + 1 >= argc) {
            tlogf(stdout, "[test-app] MISSING_ARG %s", action);
            return 2;
        }
    } else {
        tlogf(stdout, "[test-app] UNKNOWN_ACTION %s", action);
        return 2;
    }

    if (strcmp(action, "new") == 0) {
        return act_new(argv[args_at]);
    }
    if (strcmp(action, "overwrite") == 0) {
        return act_overwrite(argv[args_at]);
    }
    if (strcmp(action, "append") == 0) {
        return act_append(argv[args_at]);
    }
    if (strcmp(action, "read") == 0) {
        return act_read(argv[args_at]);
    }
    if (strcmp(action, "list") == 0) {
        return act_list(argv[args_at]);
    }
    if (strcmp(action, "delete") == 0) {
        return act_delete(argv[args_at]);
    }
    if (strcmp(action, "rename") == 0) {
        return act_rename(argv[args_at], argv[args_at + 1]);
    }
    return act_exists(argv[args_at]);
}
