/*
 * reg_mode.c — registry-redirect tests (stage-3 acceptance, W3T13).
 *
 * // allow: SIZE_OK — one cohesive module: 6 mechanically-parallel registry
 * // actions (same marker/error contract, shared helpers, one dispatch)
 * // behind a single mode, per codebase one-mode-per-file convention.
 *
 * One action per invocation (single-action contract):
 *
 *   test-app.exe reg --action create-key --path Software\GVHD_Test\SubKey
 *   test-app.exe reg --action set-value  --path Software\GVHD_Test
 *   ...
 *
 * The action is named via --action <name> (or as the first positional); the
 * target key is given via --path <rel>, where <rel> is relative to HKCU\
 * and must NOT carry the "HKCU\" prefix (e.g. "Software\GVHD_Test\SubKey").
 * Markers always echo the key with the "HKCU\" prefix prepended for
 * readability, so the assert script can rebuild expected strings by
 * prefixing its own input path.
 *
 * All registry access uses the wide-char Win32 APIs (Reg*W). The sandbox
 * hook (W3T12, parallel) rewrites HKCU\Software\... into the per-game hive
 * HKU\GameVHD_<id>\Software\... with read-through, so this mode never knows
 * a redirect happened — it just performs the Win32 call on the given path.
 * delete-key uses RegDeleteKeyExW with KEY_WOW64_64KEY so the x86 build
 * deletes the same 64-bit-view key the x64 build creates (on x64 the flag
 * selects the identity view; a plain RegDeleteKeyW would fail with
 * ERROR_ACCESS_DENIED on a non-empty key because it is not recursive).
 *
 * Output contract (assert_reg.ps1, W3T15, regex-matches these exact
 * markers; every line goes through tlogf() so it lands BOTH on stdout and
 * in the --log file; <path> below is the marker form "HKCU\<rel>"):
 *
 *   [test-app] REG_MODE_START
 *   [test-app] REG_CREATE_KEY <path> OK|EXISTS|ERR=<code>
 *   [test-app] REG_SET_VALUE <path> OK|ERR=<code>
 *   [test-app] REG_READ_VALUE <path> VALUE=<data>|NOT_FOUND|ERR=<code>
 *   [test-app] REG_ENUM <path> KEY=<name>                   (one per subkey, sorted)
 *   [test-app] REG_ENUM <path> VALUE=<name>=<data>          (one per value, sorted by name)
 *   [test-app] REG_ENUM <path> NOT_FOUND|ERR=<code>         (key missing / enum failed)
 *   [test-app] REG_ENUM <path> ERR=toobig                   (entry beyond fixed caps)
 *   [test-app] REG_DELETE_VALUE <path> OK|NOT_FOUND|ERR=<code>
 *   [test-app] REG_DELETE_KEY <path> OK|NOT_FOUND|ERR=<code>
 *   [test-app] UNKNOWN_ACTION <name>                        (exit 2)
 *   [test-app] MISSING_ARG <action|path>                    (exit 2)
 *   [test-app] UNKNOWN_ARG <arg>                            (exit 2)
 *
 * Enumerate caps (fixed buffers at the registry limits): subkey names up to
 * 255 chars, value names up to 16383 chars, value data up to 4096 bytes;
 * anything larger prints REG_ENUM ERR=toobig (exit 1).
 *
 * Value data: set-value writes the REG_SZ value GVHDTestValue =
 * "GVHD_TEST_VALUE_<pid>"; read-value / delete-value act on that same
 * fixed value name. <data> is the REG_SZ content converted back to ANSI
 * (CP_ACP); a value of an unexpected (non-string) type prints
 * VALUE=<binary> and counts as a failure. Non-string values found during
 * enumerate print VALUE=<name>=<binary>.
 *
 * Exit code: 0 = every outcome OK; 1 = any outcome was EXISTS/NOT_FOUND/
 * ERR=/VALUE=<binary>; 2 = usage error (unknown action, missing arg).
 *
 * Determinism: REG_ENUM lines are sorted by name (wcscmp) regardless of
 * the hive's enumeration order; the only varying token inside markers is
 * the process ID embedded in the value data.
 *
 * Semantics: create-key = RegCreateKeyExW (EXISTS when the key already
 * exists); set-value = RegCreateKeyExW open-or-create then RegSetValueExW
 * (auto-creates the leaf key when missing); read-value / enumerate =
 * RegOpenKeyExW; delete-value = RegDeleteValueW; delete-key =
 * RegDeleteKeyExW (deletes the key and all its subkeys, recursive).
 * NOT_FOUND covers both a missing key (ERROR_FILE_NOT_FOUND /
 * ERROR_PATH_NOT_FOUND) and, for read-value / delete-value, a missing
 * value (ERROR_FILE_NOT_FOUND).
 *
 * Windows-only: this file builds only with the mingw-w64 cross-toolchains
 * (see Makefile) and runs only on Windows; it is never built on Linux.
 */
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <windows.h>
#include <wchar.h>

#include "test_app_log.h"

/* Fixed value name used by set-value / read-value / delete-value. */
#define VALUE_NAME L"GVHDTestValue"

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

/* Convert REG_SZ/REG_EXPAND_SZ data (`size` raw bytes of wide chars) into a
 * NUL-terminated ANSI heap string. Returns NULL when the data is not a
 * convertible wide string (caller treats it as binary). */
static char *regdata_to_ansi(const BYTE *data, DWORD size)
{
    if (size % sizeof(wchar_t) != 0) {
        return NULL;
    }
    int nchars = (int)(size / sizeof(wchar_t));
    if (nchars == 0) {
        return strdup("");
    }
    int need = WideCharToMultiByte(CP_ACP, 0, (const wchar_t *)data, nchars,
                                   NULL, 0, NULL, NULL);
    if (need <= 0) {
        return NULL;
    }
    char *s = malloc((size_t)need + 1);
    if (s == NULL) {
        return NULL;
    }
    if (WideCharToMultiByte(CP_ACP, 0, (const wchar_t *)data, nchars, s, need,
                            NULL, NULL) != need) {
        free(s);
        return NULL;
    }
    s[need] = '\0';
    return s;
}

/* One marker line: "[test-app] <action> HKCU\<path> <rest>". */
static void rmarker(const char *action, const char *path, const char *fmt, ...)
{
    char buf[4096];
    va_list ap;

    va_start(ap, fmt);
    vsnprintf(buf, sizeof(buf), fmt, ap);
    va_end(ap);

    tlogf(stdout, "[test-app] %s HKCU\\%s %s", action, path, buf);
}

static int act_create_key(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        rmarker("REG_CREATE_KEY", path, "ERR=oom");
        return 1;
    }
    HKEY hk;
    DWORD disp = 0;
    LONG st = RegCreateKeyExW(HKEY_CURRENT_USER, wp, 0, NULL,
                              REG_OPTION_NON_VOLATILE,
                              KEY_CREATE_SUB_KEY | KEY_SET_VALUE, NULL, &hk,
                              &disp);
    free(wp);
    if (st != ERROR_SUCCESS) {
        rmarker("REG_CREATE_KEY", path, "ERR=%lu", (unsigned long)st);
        return 1;
    }
    RegCloseKey(hk);
    if (disp == REG_OPENED_EXISTING_KEY) {
        rmarker("REG_CREATE_KEY", path, "EXISTS");
        return 1;
    }
    rmarker("REG_CREATE_KEY", path, "OK");
    return 0;
}

static int act_set_value(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        rmarker("REG_SET_VALUE", path, "ERR=oom");
        return 1;
    }
    HKEY hk;
    LONG st = RegCreateKeyExW(HKEY_CURRENT_USER, wp, 0, NULL,
                              REG_OPTION_NON_VOLATILE, KEY_SET_VALUE, NULL,
                              &hk, NULL);
    free(wp);
    if (st != ERROR_SUCCESS) {
        rmarker("REG_SET_VALUE", path, "ERR=%lu", (unsigned long)st);
        return 1;
    }
    wchar_t wval[64];
    int n = swprintf(wval, 64, L"GVHD_TEST_VALUE_%lu",
                     (unsigned long)GetCurrentProcessId());
    LONG s2;
    if (n < 0) {
        s2 = ERROR_BAD_FORMAT;
    } else {
        s2 = RegSetValueExW(hk, VALUE_NAME, 0, REG_SZ, (const BYTE *)wval,
                            (DWORD)(n + 1) * sizeof(wchar_t));
    }
    RegCloseKey(hk);
    if (s2 != ERROR_SUCCESS) {
        rmarker("REG_SET_VALUE", path, "ERR=%lu", (unsigned long)s2);
        return 1;
    }
    rmarker("REG_SET_VALUE", path, "OK");
    return 0;
}

static int act_read_value(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        rmarker("REG_READ_VALUE", path, "ERR=oom");
        return 1;
    }
    HKEY hk;
    LONG st = RegOpenKeyExW(HKEY_CURRENT_USER, wp, 0, KEY_QUERY_VALUE, &hk);
    free(wp);
    if (st != ERROR_SUCCESS) {
        if (st == ERROR_FILE_NOT_FOUND || st == ERROR_PATH_NOT_FOUND) {
            rmarker("REG_READ_VALUE", path, "NOT_FOUND");
        } else {
            rmarker("REG_READ_VALUE", path, "ERR=%lu", (unsigned long)st);
        }
        return 1;
    }

    DWORD type = 0;
    DWORD size = 0;
    st = RegQueryValueExW(hk, VALUE_NAME, NULL, &type, NULL, &size);
    if (st == ERROR_FILE_NOT_FOUND || st == ERROR_PATH_NOT_FOUND) {
        rmarker("REG_READ_VALUE", path, "NOT_FOUND");
        RegCloseKey(hk);
        return 1;
    }
    if (st != ERROR_SUCCESS) {
        rmarker("REG_READ_VALUE", path, "ERR=%lu", (unsigned long)st);
        RegCloseKey(hk);
        return 1;
    }

    int rc = 0;
    if (type != REG_SZ && type != REG_EXPAND_SZ) {
        rmarker("REG_READ_VALUE", path, "VALUE=<binary>");
        rc = 1;
    } else {
        BYTE *buf = malloc(size > 0 ? (size_t)size : 2);
        if (buf == NULL) {
            rmarker("REG_READ_VALUE", path, "ERR=oom");
            rc = 1;
        } else {
            DWORD got = size;
            st = RegQueryValueExW(hk, VALUE_NAME, NULL, &type, buf, &got);
            if (st != ERROR_SUCCESS) {
                rmarker("REG_READ_VALUE", path, "ERR=%lu", (unsigned long)st);
                rc = 1;
            } else {
                char *s = regdata_to_ansi(buf, got);
                if (s == NULL) {
                    rmarker("REG_READ_VALUE", path, "VALUE=<binary>");
                    rc = 1;
                } else {
                    rmarker("REG_READ_VALUE", path, "VALUE=%s", s);
                    free(s);
                }
            }
            free(buf);
        }
    }
    RegCloseKey(hk);
    return rc;
}

static int wcscmp_qsort(const void *a, const void *b)
{
    const wchar_t *pa = *(const wchar_t *const *)a;
    const wchar_t *pb = *(const wchar_t *const *)b;
    return wcscmp(pa, pb);
}

typedef struct {
    wchar_t *name; /* NUL-terminated copy */
} reg_subkey;

typedef struct {
    wchar_t *name; /* NUL-terminated copy */
    DWORD type;
    BYTE *data; /* raw value data, `size` bytes (NULL when size == 0) */
    DWORD size;
} reg_value;

/* Wide name -> ANSI into `buf`; returns "<unconvertible>" on failure. */
static const char *wansi(const wchar_t *w, char *buf, size_t cap)
{
    if (WideCharToMultiByte(CP_ACP, 0, w, -1, buf, (int)cap, NULL, NULL) <= 0) {
        return "<unconvertible>";
    }
    return buf;
}

/* Enumeration buffers at the registry component limits: subkey names are
 * limited to 255 chars, value names to 16383 chars. Anything larger (or
 * value data beyond the cap) prints ERR=toobig and stops. */
#define ENUM_KEY_BUF  256
#define ENUM_NAME_BUF 16384
#define ENUM_DATA_BUF 4096

static int act_enumerate(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        rmarker("REG_ENUM", path, "ERR=oom");
        return 1;
    }
    HKEY hk;
    LONG st = RegOpenKeyExW(HKEY_CURRENT_USER, wp, 0, KEY_READ, &hk);
    free(wp);
    if (st != ERROR_SUCCESS) {
        if (st == ERROR_FILE_NOT_FOUND || st == ERROR_PATH_NOT_FOUND) {
            rmarker("REG_ENUM", path, "NOT_FOUND");
        } else {
            rmarker("REG_ENUM", path, "ERR=%lu", (unsigned long)st);
        }
        return 1;
    }

    reg_subkey *keys = NULL;
    reg_value *vals = NULL;
    size_t nkeys = 0;
    size_t nvals = 0;
    int rc = 0;
    wchar_t kname[ENUM_KEY_BUF];
    wchar_t vname[ENUM_NAME_BUF];
    BYTE vdata[ENUM_DATA_BUF];

    DWORD kidx = 0;
    for (;;) {
        LONG s2 = RegEnumKeyW(hk, kidx, kname, ENUM_KEY_BUF);
        if (s2 == ERROR_NO_MORE_ITEMS) {
            break;
        }
        if (s2 == ERROR_MORE_DATA) {
            rmarker("REG_ENUM", path, "ERR=toobig");
            rc = 1;
            break;
        }
        if (s2 != ERROR_SUCCESS) {
            rmarker("REG_ENUM", path, "ERR=%lu", (unsigned long)s2);
            rc = 1;
            break;
        }
        reg_subkey *nk = realloc(keys, (nkeys + 1) * sizeof(*nk));
        if (nk == NULL) {
            rmarker("REG_ENUM", path, "ERR=oom");
            rc = 1;
            break;
        }
        keys = nk;
        keys[nkeys].name = wcsdup(kname);
        if (keys[nkeys].name == NULL) {
            rmarker("REG_ENUM", path, "ERR=oom");
            rc = 1;
            break;
        }
        nkeys++;
        kidx++;
    }

    if (rc == 0) {
        DWORD vidx = 0;
        for (;;) {
            DWORD ncap = ENUM_NAME_BUF;
            DWORD dcap = ENUM_DATA_BUF;
            DWORD vtype = 0;
            LONG s2 = RegEnumValueW(hk, vidx, vname, &ncap, NULL, &vtype,
                                    vdata, &dcap);
            if (s2 == ERROR_NO_MORE_ITEMS) {
                break;
            }
            if (s2 == ERROR_MORE_DATA) {
                rmarker("REG_ENUM", path, "ERR=toobig");
                rc = 1;
                break;
            }
            if (s2 != ERROR_SUCCESS) {
                rmarker("REG_ENUM", path, "ERR=%lu", (unsigned long)s2);
                rc = 1;
                break;
            }
            reg_value *nv = realloc(vals, (nvals + 1) * sizeof(*nv));
            if (nv == NULL) {
                rmarker("REG_ENUM", path, "ERR=oom");
                rc = 1;
                break;
            }
            vals = nv;
            reg_value *v = &vals[nvals];
            v->name = wcsdup(vname);
            v->data = dcap > 0 ? malloc((size_t)dcap) : NULL;
            if (v->name == NULL || (dcap > 0 && v->data == NULL)) {
                free(v->name);
                free(v->data);
                rmarker("REG_ENUM", path, "ERR=oom");
                rc = 1;
                break;
            }
            if (dcap > 0) {
                memcpy(v->data, vdata, (size_t)dcap);
            }
            v->type = vtype;
            v->size = dcap;
            nvals++;
            vidx++;
        }
    }

    if (rc == 0) {
        qsort(keys, nkeys, sizeof(*keys), wcscmp_qsort);
        qsort(vals, nvals, sizeof(*vals), wcscmp_qsort);

        char nb[1024];
        size_t i;
        for (i = 0; i < nkeys; i++) {
            rmarker("REG_ENUM", path, "KEY=%s",
                    wansi(keys[i].name, nb, sizeof(nb)));
        }
        for (i = 0; i < nvals; i++) {
            const char *name = wansi(vals[i].name, nb, sizeof(nb));
            char *ds = (vals[i].type == REG_SZ || vals[i].type == REG_EXPAND_SZ)
                           ? regdata_to_ansi(vals[i].data, vals[i].size)
                           : NULL;
            if (ds != NULL) {
                rmarker("REG_ENUM", path, "VALUE=%s=%s", name, ds);
                free(ds);
            } else {
                rmarker("REG_ENUM", path, "VALUE=%s=<binary>", name);
            }
        }
    }

    size_t i;
    for (i = 0; i < nkeys; i++) {
        free(keys[i].name);
    }
    free(keys);
    for (i = 0; i < nvals; i++) {
        free(vals[i].name);
        free(vals[i].data);
    }
    free(vals);
    RegCloseKey(hk);
    return rc;
}

static int act_delete_value(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        rmarker("REG_DELETE_VALUE", path, "ERR=oom");
        return 1;
    }
    HKEY hk;
    LONG st = RegOpenKeyExW(HKEY_CURRENT_USER, wp, 0, KEY_SET_VALUE, &hk);
    free(wp);
    if (st != ERROR_SUCCESS) {
        if (st == ERROR_FILE_NOT_FOUND || st == ERROR_PATH_NOT_FOUND) {
            rmarker("REG_DELETE_VALUE", path, "NOT_FOUND");
        } else {
            rmarker("REG_DELETE_VALUE", path, "ERR=%lu", (unsigned long)st);
        }
        return 1;
    }
    st = RegDeleteValueW(hk, VALUE_NAME);
    RegCloseKey(hk);
    if (st == ERROR_FILE_NOT_FOUND || st == ERROR_PATH_NOT_FOUND) {
        rmarker("REG_DELETE_VALUE", path, "NOT_FOUND");
        return 1;
    }
    if (st != ERROR_SUCCESS) {
        rmarker("REG_DELETE_VALUE", path, "ERR=%lu", (unsigned long)st);
        return 1;
    }
    rmarker("REG_DELETE_VALUE", path, "OK");
    return 0;
}

static int act_delete_key(const char *path)
{
    wchar_t *wp = to_wide(path);
    if (wp == NULL) {
        rmarker("REG_DELETE_KEY", path, "ERR=oom");
        return 1;
    }
    LONG st = RegDeleteKeyExW(HKEY_CURRENT_USER, wp, KEY_WOW64_64KEY, 0);
    free(wp);
    if (st == ERROR_FILE_NOT_FOUND || st == ERROR_PATH_NOT_FOUND) {
        rmarker("REG_DELETE_KEY", path, "NOT_FOUND");
        return 1;
    }
    if (st != ERROR_SUCCESS) {
        rmarker("REG_DELETE_KEY", path, "ERR=%lu", (unsigned long)st);
        return 1;
    }
    rmarker("REG_DELETE_KEY", path, "OK");
    return 0;
}

static void print_action_help(void)
{
    tlogf(stdout,
          "[test-app] reg mode actions (usage: test-app reg [--action <name>] [--path <rel>]):");
    tlogf(stdout, "  create-key    create a key (EXISTS if already present)");
    tlogf(stdout, "  set-value     write REG_SZ GVHDTestValue = GVHD_TEST_VALUE_<pid>;");
    tlogf(stdout, "                auto-creates the leaf key when missing");
    tlogf(stdout, "  read-value    read GVHDTestValue as VALUE=<data>");
    tlogf(stdout, "  enumerate     list subkeys (KEY=<name>) and values (VALUE=<name>=<data>),");
    tlogf(stdout, "                sorted by name");
    tlogf(stdout, "  delete-value  delete GVHDTestValue");
    tlogf(stdout, "  delete-key    delete the key tree (64-bit view, recursive)");
    tlogf(stdout, "  --path is relative to HKCU\\ (no prefix); markers print HKCU\\<rel>");
    tlogf(stdout, "  exit: 0 all OK, 1 any EXISTS/NOT_FOUND/ERR, 2 usage error");
}

int reg_mode(int argc, char **argv)
{
    tlogf(stdout, "[test-app] REG_MODE_START");

    const char *action = NULL;
    const char *path = NULL;
    int i;

    for (i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--help") == 0) {
            print_action_help();
            return 0;
        }
        if (strcmp(argv[i], "--action") == 0) {
            if (i + 1 >= argc) {
                tlogf(stdout, "[test-app] MISSING_ARG action");
                return 2;
            }
            action = argv[++i];
        } else if (strcmp(argv[i], "--path") == 0) {
            if (i + 1 >= argc) {
                tlogf(stdout, "[test-app] MISSING_ARG path");
                return 2;
            }
            path = argv[++i];
        } else if (argv[i][0] == '-') {
            tlogf(stdout, "[test-app] UNKNOWN_ARG %s", argv[i]);
            return 2;
        } else if (action == NULL) {
            action = argv[i];
        } else if (path == NULL) {
            path = argv[i];
        } else {
            tlogf(stdout, "[test-app] UNKNOWN_ARG %s", argv[i]);
            return 2;
        }
    }

    if (action == NULL) {
        tlogf(stdout, "[test-app] MISSING_ARG action");
        return 2;
    }
    if (path == NULL) {
        tlogf(stdout, "[test-app] MISSING_ARG path");
        return 2;
    }

    if (strcmp(action, "create-key") == 0) {
        return act_create_key(path);
    }
    if (strcmp(action, "set-value") == 0) {
        return act_set_value(path);
    }
    if (strcmp(action, "read-value") == 0) {
        return act_read_value(path);
    }
    if (strcmp(action, "enumerate") == 0) {
        return act_enumerate(path);
    }
    if (strcmp(action, "delete-value") == 0) {
        return act_delete_value(path);
    }
    if (strcmp(action, "delete-key") == 0) {
        return act_delete_key(path);
    }
    tlogf(stdout, "[test-app] UNKNOWN_ACTION %s", action);
    return 2;
}
