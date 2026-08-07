/*
 * inject_mode.c — hook-DLL injection self-check (skeleton).
 *
 * Future: print HOOK_DLL_PRESENT marker when gvhook is injected, exercise
 * the parent/child process tree coverage, then return a pass/fail code.
 */
#include <stdio.h>

#include "test_app_log.h"

int inject_mode(int argc, char **argv)
{
    (void)argc;
    (void)argv;
    tlogf(stdout, "MODE_NOT_IMPLEMENTED inject");
    return 3;
}
