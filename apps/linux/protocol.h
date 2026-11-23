#pragma once
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/resource.h>
#include <unistd.h>

/* stdin: message, expiry, then names, all NUL terminated.
   fd 3: ready byte, then submitted values in the same order, NUL terminated.
   Exit 2 means cancelled. No diagnostics contain field contents. */
static FILE *result_stream(void) {
    struct rlimit limit = {0, 0};
    setrlimit(RLIMIT_CORE, &limit);
    if (fcntl(3, F_SETFD, FD_CLOEXEC) == -1) return NULL;
    return fdopen(3, "w");
}

static char *next_string(void) {
    char *text = NULL;
    size_t size = 0;
    ssize_t n = getdelim(&text, &size, 0, stdin);
    if (n <= 0 || text[n - 1] != 0) {
        free(text);
        return NULL;
    }
    return text;
}

static int signal_ready(FILE *out) {
    return fputc('R', out) != EOF && fflush(out) == 0;
}
