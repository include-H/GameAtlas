/*
 * test_app_log.c — shared logging helper implementation.
 *
 * Each tlogf() call emits one timestamped line to the given stream and, when
 * the log file is open, an identical line to the log file.
 */
#include "test_app_log.h"

#include <stdarg.h>
#include <string.h>
#include <time.h>

static FILE *g_log_file = NULL;

static void emit_timestamped(FILE *out, const char *fmt, va_list ap)
{
    char ts[32];
    time_t now;
    struct tm tmv;

    now = time(NULL);
    localtime_s(&tmv, &now);
    strftime(ts, sizeof(ts), "%Y-%m-%d %H:%M:%S", &tmv);

    fprintf(out, "[%s] ", ts);
    vfprintf(out, fmt, ap);
    fputc('\n', out);
    fflush(out);
}

int log_init(const char *path)
{
    if (path == NULL) {
        path = "test-app.log";
    }
    g_log_file = fopen(path, "a");
    return g_log_file != NULL ? 0 : -1;
}

void log_close(void)
{
    if (g_log_file != NULL) {
        fclose(g_log_file);
        g_log_file = NULL;
    }
}

void tlogf(FILE *out, const char *fmt, ...)
{
    va_list ap;

    va_start(ap, fmt);
    emit_timestamped(out, fmt, ap);

    if (g_log_file != NULL) {
        va_list ap2;
        va_copy(ap2, ap);
        emit_timestamped(g_log_file, fmt, ap2);
        va_end(ap2);
    }

    va_end(ap);
}
