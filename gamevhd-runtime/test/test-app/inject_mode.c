/*
 * inject_mode.c — hook-DLL injection self-check (stage-1 acceptance).
 *
 * The runtime injects gvhook.dll into this process (protocol §5); the hook's
 * proc.c then auto-injects every child process the process tree spawns
 * (protocol §6). This mode probes whether gvhook is actually loaded
 * in-process: GetModuleHandleW(L"gvhook") returns non-NULL exactly when the
 * DLL was injected and its image is mapped.
 *
 * Output contract (assert_inject.ps1 parses these exact [test-app] markers;
 * every line goes through tlogf() so it lands BOTH on stdout and in the
 * --log file):
 *
 *   [test-app] INJECT_SELFCHECK_START
 *   [test-app] PID <pid>
 *   [test-app] HOOK_DLL_PRESENT      (exit 0)
 *   [test-app] NOT_INJECTED          (exit 1)
 *
 * self_check() is shared with child_mode.c so every level of a child chain
 * emits the byte-identical marker trio (one definition, no drift).
 */
#include <stdio.h>
#include <windows.h>

#include "test_app_log.h"

/* Print the self-check marker trio and return the exit code:
 * 0 = injected (HOOK_DLL_PRESENT), 1 = not injected (NOT_INJECTED). */
int self_check(void)
{
    tlogf(stdout, "[test-app] INJECT_SELFCHECK_START");
    tlogf(stdout, "[test-app] PID %lu", (unsigned long)GetCurrentProcessId());

    if (GetModuleHandleW(L"gvhook") != NULL) {
        tlogf(stdout, "[test-app] HOOK_DLL_PRESENT");
        return 0;
    }
    tlogf(stdout, "[test-app] NOT_INJECTED");
    return 1;
}

int inject_mode(int argc, char **argv)
{
    (void)argc;
    (void)argv;
    return self_check();
}
