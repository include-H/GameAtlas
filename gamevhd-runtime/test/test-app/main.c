/*
 * main.c — test-app entry point: mode dispatch for the GameVHD Runtime
 * sandbox test program.
 *
 * Usage: test-app.exe [--log <path>] <mode> [mode args...]
 *
 *   --log <path>   log file for duplicated timestamped output
 *                  (default: test-app.log in CWD)
 *   inject         hook-DLL injection self-check (future)
 *   file           file-redirect operations (future)
 *   reg            registry-redirect operations (future)
 *   child          child-process injection propagation (future)
 *
 * Modes forward their remaining argv; each entry receives
 * (argc', argv') with argv'[0] = mode name, argv'[1..] = mode args.
 */
#include <stdio.h>
#include <string.h>

#include "test_app_log.h"

int inject_mode(int argc, char **argv);
int file_mode(int argc, char **argv);
int reg_mode(int argc, char **argv);
int child_mode(int argc, char **argv);

static void usage(FILE *out)
{
    fprintf(out,
            "usage: test-app.exe [--log <path>] <mode> [mode args...]\n"
            "modes: inject | file | reg | child\n"
            "       (run with --help for per-mode details)\n");
}

static void print_help(void)
{
    printf(
        "GameVHD Runtime sandbox test program (x64/x86)\n"
        "\n"
        "usage: test-app.exe [--log <path>] <mode> [mode args...]\n"
        "\n"
        "options:\n"
        "  --log <path>   duplicate timestamped output to this log file\n"
        "                 (default: test-app.log in CWD)\n"
        "  --help         show this help\n"
        "\n"
        "modes:\n"
        "  inject   Hook-DLL presence self-check: print HOOK_DLL_PRESENT\n"
        "           marker when gvhook is injected; verify parent/child\n"
        "           injection coverage. (skeleton)\n"
        "  file     File-redirect tests: create / overwrite / append /\n"
        "           read / list / delete / rename under redirected\n"
        "           directories. (skeleton)\n"
        "  reg      Registry-redirect tests: create key / write / read /\n"
        "           enumerate / delete value / delete key under\n"
        "           redirected HKCU paths, incl. Wow6432Node for x86.\n"
        "           (skeleton)\n"
        "  child    Child-process injection propagation: spawn a child\n"
        "           test-app and verify the hook DLL covers the whole\n"
        "           process tree. (skeleton)\n");
}

int main(int argc, char **argv)
{
    const char *log_path = NULL;
    const char *mode;
    int restc, i, j;
    int rc;

    /* Scan args: drop --log <path>, handle --help, compact the rest. */
    for (i = 1, j = 1; i < argc; i++) {
        if (strcmp(argv[i], "--help") == 0) {
            print_help();
            return 0;
        }
        if (strcmp(argv[i], "--log") == 0 && i + 1 < argc) {
            log_path = argv[++i];
            continue;
        }
        argv[j++] = argv[i];
    }
    restc = j - 1; /* remaining args live in argv[1..j-1] */

    if (restc == 0) {
        usage(stderr);
        return 2;
    }

    if (log_init(log_path) != 0) {
        fprintf(stderr, "warning: cannot open log file '%s', "
                        "logging to file disabled\n",
                log_path != NULL ? log_path : "test-app.log");
    }

    mode = argv[1];
    if (strcmp(mode, "inject") == 0) {
        rc = inject_mode(restc, argv + 1);
    } else if (strcmp(mode, "file") == 0) {
        rc = file_mode(restc, argv + 1);
    } else if (strcmp(mode, "reg") == 0) {
        rc = reg_mode(restc, argv + 1);
    } else if (strcmp(mode, "child") == 0) {
        rc = child_mode(restc, argv + 1);
    } else {
        fprintf(stderr, "test-app: unknown mode: %s\n", mode);
        usage(stderr);
        rc = 2;
    }

    log_close();
    return rc;
}
