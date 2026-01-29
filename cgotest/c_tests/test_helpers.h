#define _POSIX_C_SOURCE 200809L

#ifndef TEST_HELPERS_H
#define TEST_HELPERS_H

#include <assert.h>
#include <inttypes.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <time.h>

#include "ygrpc_cgo_common.h"

// Forward declaration for test_helpers.h users that include this header before
// libygrpc.h.
//
// Note: Use explicit builtin types to match cgo-generated typedefs:
//   GoUint64 -> unsigned long long
//   GoInt    -> long long
extern unsigned long long Ygrpc_GetErrorMsg(
    unsigned long long errorID,
    void** msgPtr,
    long long* msgLen,
    void** msgFree
);

/**
 * @brief Print error message to stderr and abort.
 * 
 * Usage:
 *   YGRPC_FAILF("expected non-zero error id\n");
 *   YGRPC_FAILF("unexpected value: %d\n", value);
 */
#define YGRPC_FAILF(fmt, ...) do { \
    fprintf(stderr, "%s:%d: ", __FILE__, __LINE__); \
    fprintf(stderr, fmt, ##__VA_ARGS__); \
    abort(); \
} while(0)

/**
 * @brief Assert that condition is true; fail with message if not.
 * 
 * Usage:
 *   YGRPC_ASSERT(ptr != NULL);
 */
#define YGRPC_ASSERT(cond) do { \
    if (!(cond)) { \
        YGRPC_FAILF("assertion failed: %s\n", #cond); \
    } \
} while(0)

/**
 * @brief Assert with custom failure message.
 * 
 * Usage:
 *   YGRPC_ASSERTF(err_id == 0, "error id should be 0, got %d\n", err_id);
 */
#define YGRPC_ASSERTF(cond, fmt, ...) do { \
    if (!(cond)) { \
        YGRPC_FAILF(fmt, ##__VA_ARGS__); \
    } \
} while(0)

/**
 * @brief Compare two strings with lengths.
 * 
 * Checks both length and memcmp. Fails with detailed message if not equal.
 * 
 * Usage:
 *   ygrpc_expect_eq_str(out_msg, out_len, "expected output");
 */
static inline void ygrpc_expect_eq_str(const char* got, int got_len, const char* want) {
    int want_len = (int)strlen(want);
    if (got_len != want_len || memcmp(got, want, (size_t)want_len) != 0) {
        fprintf(stderr, "expected '%s' (%d), got '%.*s' (%d)\n", 
                want, want_len, got_len, got, got_len);
        abort();
    }
}

/**
 * @brief Check that pointer is not NULL.
 * 
 * Fails with descriptive message if pointer is NULL.
 * 
 * Usage:
 *   ygrpc_expect_ptr(resp_ptr, "response buffer");
 */
static inline void ygrpc_expect_ptr(const void* p, const char* what) {
    if (p == NULL) {
        YGRPC_FAILF("expected non-NULL %s\n", what);
    }
}

/**
 * @brief Check that error ID is 0 (no error).
 * 
 * Fails with message if err_id is not 0.
 * 
 * Usage:
 *   ygrpc_expect_err0_i64(err_id, "RPC call");
 */
void ygrpc_expect_err0_i64(uint64_t err_id, const char* what);

/**
 * @brief Poll a flag until it becomes true or timeout.
 * 
 * Useful for waiting on async operations (e.g., bidi stream completion).
 * Calls nanosleep between iterations.
 * 
 * Usage:
 *   volatile int done = 0;
 *   // ... start async operation that sets done=1 ...
 *   ygrpc_wait_done_flag(&done, 200, 10*1000*1000);  // 200 iterations, 10ms each
 */
static inline void ygrpc_wait_done_flag(const volatile int* done_flag, int max_iters, long sleep_ns) {
    for (int i = 0; i < max_iters; i++) {
        if (*done_flag) {
            return;
        }
        struct timespec ts;
        ts.tv_sec = sleep_ns / (1000 * 1000 * 1000);
        ts.tv_nsec = sleep_ns % (1000 * 1000 * 1000);
        nanosleep(&ts, NULL);
    }
}

#endif  // TEST_HELPERS_H
