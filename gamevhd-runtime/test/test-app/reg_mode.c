/*
 * reg_mode.c — registry-redirect tests (skeleton).
 *
 * Future: reg --action <建键|写值|读值|枚举|删值|删键> --path <HKCU\...>
 * Verify writes land in the sandboxed hive, incl. Wow6432Node for x86.
 */
#include <stdio.h>

#include "test_app_log.h"

int reg_mode(int argc, char **argv)
{
    (void)argc;
    (void)argv;
    tlogf(stdout, "MODE_NOT_IMPLEMENTED reg");
    return 3;
}
