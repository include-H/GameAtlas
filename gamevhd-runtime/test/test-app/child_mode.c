/*
 * child_mode.c — child-process injection propagation (skeleton).
 *
 * Future: spawn a child test-app (inject mode) and verify the hook DLL
 * covers the whole process tree; propagate pass/fail from the child.
 */
#include <stdio.h>

#include "test_app_log.h"

int child_mode(int argc, char **argv)
{
    (void)argc;
    (void)argv;
    tlogf(stdout, "MODE_NOT_IMPLEMENTED child");
    return 3;
}
