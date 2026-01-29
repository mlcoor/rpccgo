#define _POSIX_C_SOURCE 200809L

#include "test_helpers.h"
#include "ygrpc_cgo_common.h"

#include <stdio.h>
#include <stdlib.h>
#include <inttypes.h>

// Forward declaration from libygrpc.h (CGO-generated)
// Use explicit builtin types to match cgo typedefs:
//   GoUint64 -> unsigned long long
//   GoInt    -> long long
extern unsigned long long Ygrpc_GetErrorMsg(
    unsigned long long errorID,
    void** msgPtr,
    long long* msgLen,
    void** msgFree
);

/**
 * @brief Check that error ID is 0 (no error).
 * 
 * If err_id is non-zero:
 * - Calls Ygrpc_GetErrorMsg to retrieve the error message
 * - Prints the error message to stderr
 * - Frees the error message memory
 * - Aborts the program
 * 
 * Usage:
 *   ygrpc_expect_err0_i64(err_id, "RPC call");
 */
void ygrpc_expect_err0_i64(uint64_t err_id, const char* what) {
    if (err_id != 0) {
        void* emsg_ptr = NULL;
        long long emsg_len = 0;
        void* emsg_free = NULL;

        unsigned long long rc = Ygrpc_GetErrorMsg(
            (unsigned long long)err_id,
            &emsg_ptr,
            &emsg_len,
            &emsg_free
        );

        fprintf(stderr, "%s:%d: ", __FILE__, __LINE__);
        fprintf(stderr, "expected %s to succeed, got error %" PRIu64, what, err_id);

        if (rc == 0 && emsg_ptr != NULL && emsg_len > 0) {
            fprintf(stderr, ": ");
            fwrite(emsg_ptr, 1, (size_t)emsg_len, stderr);
        } else {
            fprintf(
                stderr,
                " (Ygrpc_GetErrorMsg failed: rc=%llu len=%lld ptr=%p free=%p)",
                rc,
                emsg_len,
                emsg_ptr,
                emsg_free
            );
        }
        fprintf(stderr, "\n");

        // emsg_free comes back as void* (see cgo header); follow existing test
        // pattern to cast and call.
        if (emsg_ptr != NULL) {
            call_free_func((FreeFunc)emsg_free, emsg_ptr);
        }

        abort();
    }
}
