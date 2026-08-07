/*
 * test_app_log.h — shared logging helper for test-app.
 *
 * Log lines are timestamped and written BOTH to the caller-supplied stream
 * (typically stdout / stderr) AND to the log file opened via log_init().
 */
#ifndef TEST_APP_LOG_H
#define TEST_APP_LOG_H

#include <stdio.h>

/* Open the log file for appending. Returns 0 on success, -1 on failure
 * (logging to file is then silently disabled; stdout logging keeps working).
 * Passing NULL falls back to "test-app.log" in the current directory. */
int log_init(const char *path);

/* Close the log file. Safe to call multiple times / when never opened. */
void log_close(void);

/* Timestamped line written to `out` and duplicated to the log file. */
void tlogf(FILE *out, const char *fmt, ...);

#endif /* TEST_APP_LOG_H */
