/*
 * child_mode.c — child-process injection propagation (stage-1 acceptance).
 *
 * Verifies the "process-tree coverage" invariant of protocol §6: when the
 * parent is injected, the hook's NtCreateUserProcess hook auto-injects every
 * child (CHILD_INJECTED in the hook log) BEFORE the child runs any user code,
 * so each generation's GetModuleHandleW(L"gvhook") self-check passes too.
 *
 * Each level of the chain:
 *   1. runs the shared self_check() (inject_mode.c) — prints the marker trio
 *      INJECT_SELFCHECK_START / PID / HOOK_DLL_PRESENT | NOT_INJECTED;
 *   2. if --depth N with N > 0, spawns the NEXT generation of itself
 *      (same exe, child mode, depth N-1), waits for it and exits with the
 *      child's exit code (the deepest level exits with its own self-check
 *      code, which therefore propagates to the root).
 *
 * Usage: test-app child [--depth N]
 *   --depth N   chain length = N + 1 levels (default 1; cap 100)
 *
 * The global --log <path> is consumed by main.c and never reaches this mode's
 * argv, so the log path is re-discovered from the raw command line
 * (GetCommandLineW) and forwarded to every spawned level — the whole chain
 * appends to one file, and the root's stdout is inherited by all descendants
 * (bInheritHandles) so the assert script can capture the full chain from a
 * single stream.
 *
 * Output contract (all lines [test-app]-prefixed):
 *   [test-app] SPAWN_CHILD pid=<pid> depth=<n>
 *   [test-app] CHILD_EXIT_CODE <code>
 *   [test-app] SPAWN_FAILED <reason>          (exit 2)
 *   [test-app] BAD_DEPTH / UNKNOWN_ARG ...    (exit 2)
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <windows.h>
#include <wchar.h>

#include "test_app_log.h"

int self_check(void); /* shared with inject_mode.c */

#define DEPTH_DEFAULT 1
#define DEPTH_MAX     100
#define PATH_BUF_INIT 260

/* Absolute path of this executable, heap-allocated, growing on truncation.
 * Returns NULL on failure (caller frees). */
static wchar_t *self_path_alloc(void)
{
    DWORD size = PATH_BUF_INIT;
    wchar_t *buf = NULL;

    for (;;) {
        wchar_t *nb = realloc(buf, size * sizeof(wchar_t));
        if (nb == NULL) {
            free(buf);
            return NULL;
        }
        buf = nb;
        DWORD n = GetModuleFileNameW(NULL, buf, size);
        if (n == 0) {
            free(buf);
            return NULL;
        }
        if (n < size) {
            return buf; /* fits, NUL-terminated */
        }
        size *= 2;
    }
}

/* Quote-aware scan of the raw command line for the "--log <path>" pair that
 * main.c already consumed from argv. Returns a heap copy of the path, or NULL
 * when absent. */
static wchar_t *discover_log_path(void)
{
    const wchar_t *p = GetCommandLineW();
    wchar_t tok[4096];
    int expect_path = 0;

    while (*p != L'\0') {
        while (*p == L' ') {
            p++;
        }
        if (*p == L'\0') {
            break;
        }
        size_t n = 0;
        if (*p == L'"') {
            p++;
            while (*p != L'\0' && *p != L'"') {
                if (n + 1 < sizeof(tok) / sizeof(tok[0])) {
                    tok[n++] = *p;
                }
                p++;
            }
            if (*p == L'"') {
                p++;
            }
        } else {
            while (*p != L'\0' && *p != L' ') {
                if (n + 1 < sizeof(tok) / sizeof(tok[0])) {
                    tok[n++] = *p;
                }
                p++;
            }
        }
        tok[n] = L'\0';

        if (expect_path) {
            return wcsdup(tok);
        }
        expect_path = (wcscmp(tok, L"--log") == 0);
    }
    return NULL;
}

/* Spawn the next generation (same exe, child mode, depth-1), wait for it and
 * return its exit code; 2 on any spawn/wait failure. */
static int spawn_child(const wchar_t *self, const wchar_t *log, int depth)
{
    if (wcschr(self, L'"') != NULL ||
        (log != NULL && wcschr(log, L'"') != NULL)) {
        tlogf(stdout, "[test-app] SPAWN_FAILED quote-in-path");
        return 2;
    }

    size_t need = wcslen(self) + (log != NULL ? wcslen(log) + 24 : 16) + 32;
    wchar_t *cmd = malloc(need * sizeof(wchar_t));
    if (cmd == NULL) {
        tlogf(stdout, "[test-app] SPAWN_FAILED oom");
        return 2;
    }
    int n;
    if (log != NULL) {
        n = swprintf(cmd, need, L"\"%ls\" --log \"%ls\" child --depth %d",
                     self, log, depth);
    } else {
        n = swprintf(cmd, need, L"\"%ls\" child --depth %d", self, depth);
    }
    if (n < 0) {
        free(cmd);
        tlogf(stdout, "[test-app] SPAWN_FAILED cmdline");
        return 2;
    }

    STARTUPINFOW si;
    PROCESS_INFORMATION pi;
    ZeroMemory(&si, sizeof(si));
    ZeroMemory(&pi, sizeof(pi));
    si.cb = sizeof(si);

    /* Inherit stdio so the whole chain lands in the captured stream. */
    if (!CreateProcessW(self, cmd, NULL, NULL, TRUE, 0, NULL, NULL, &si,
                        &pi)) {
        DWORD gle = GetLastError();
        tlogf(stdout, "[test-app] SPAWN_FAILED gle=%lu", (unsigned long)gle);
        free(cmd);
        return 2;
    }
    free(cmd);
    CloseHandle(pi.hThread);

    tlogf(stdout, "[test-app] SPAWN_CHILD pid=%lu depth=%d",
          (unsigned long)pi.dwProcessId, depth);

    DWORD wait = WaitForSingleObject(pi.hProcess, INFINITE);
    if (wait != WAIT_OBJECT_0) {
        tlogf(stdout, "[test-app] SPAWN_FAILED wait gle=%lu",
              (unsigned long)GetLastError());
        CloseHandle(pi.hProcess);
        return 2;
    }
    DWORD code = 0;
    GetExitCodeProcess(pi.hProcess, &code);
    CloseHandle(pi.hProcess);

    tlogf(stdout, "[test-app] CHILD_EXIT_CODE %lu", (unsigned long)code);
    return (int)code;
}

int child_mode(int argc, char **argv)
{
    int depth = DEPTH_DEFAULT;
    int i;

    for (i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--depth") == 0 && i + 1 < argc) {
            i++;
            char *end = NULL;
            long v = strtol(argv[i], &end, 10);
            if (end == argv[i] || *end != '\0' || v < 0 || v > DEPTH_MAX) {
                tlogf(stdout, "[test-app] BAD_DEPTH %s", argv[i]);
                return 2;
            }
            depth = (int)v;
        } else {
            tlogf(stdout, "[test-app] UNKNOWN_ARG %s", argv[i]);
            return 2;
        }
    }

    int rc = self_check();
    if (depth > 0) {
        wchar_t *self = self_path_alloc();
        wchar_t *log = discover_log_path();
        if (self == NULL) {
            tlogf(stdout, "[test-app] SPAWN_FAILED self-path");
            rc = 2;
        } else {
            int crc = spawn_child(self, log, depth - 1);
            if (crc != 0) {
                rc = crc;
            }
        }
        free(log);
        free(self);
    }
    return rc;
}
