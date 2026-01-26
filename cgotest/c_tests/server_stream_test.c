#include <assert.h>
#include <inttypes.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libygrpc.h"
#include "proto_helpers.h"

typedef struct {
    int done;
    uint64_t done_error_id;
    int count;
    char results[8][64];
} stream_state;

static void on_read_bytes(uint64_t call_id, void *resp_ptr, int resp_len, FreeFunc resp_free)
{
    stream_state *st = (stream_state *)(uintptr_t)call_id;

    const uint8_t* s_ptr = NULL;
    int s_len = 0;
    if (ygrpc_decode_string_field((const uint8_t*)resp_ptr, resp_len, 1, &s_ptr, &s_len) != 0) {
        snprintf(st->results[st->count++], sizeof(st->results[0]), "<decode error>");
    } else {
        int n = s_len;
        if (n > (int)sizeof(st->results[0]) - 1) n = (int)sizeof(st->results[0]) - 1;
        memcpy(st->results[st->count], s_ptr, (size_t)n);
        st->results[st->count][n] = 0;
        st->count++;
    }

    if (resp_free) resp_free(resp_ptr);
}

static void on_done(uint64_t call_id, uint64_t error_id)
{
    stream_state *st = (stream_state *)(uintptr_t)call_id;
    st->done = 1;
    st->done_error_id = error_id;
}

static void on_read_native(uint64_t call_id, void *result_ptr, int result_len, FreeFunc result_free, int32_t sequence)
{
    (void)sequence;
    stream_state *st = (stream_state *)(uintptr_t)call_id;
    int n = result_len;
    if (n > (int)sizeof(st->results[0]) - 1) n = (int)sizeof(st->results[0]) - 1;
    memcpy(st->results[st->count], result_ptr, (size_t)n);
    st->results[st->count][n] = 0;
    st->count++;
    if (result_free) result_free(result_ptr);
}

int main(void) {
    int rc = Ygrpc_SetProtocol(YGRPC_PROTOCOL_UNSET);
    if (rc != 0)
    {
        fprintf(stderr, "Ygrpc_SetProtocol failed: %d\n", rc);
        return 1;
    }

    // Binary server-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        uint8_t* req = NULL;
        int req_len = 0;
        assert(ygrpc_encode_stream_request("test", 4, 7, &req, &req_len) == 0);

        uint64_t call_id = (uint64_t)(uintptr_t)&st;
        int err_id = Ygrpc_StreamService_ServerStreamCall(req, req_len, (void *)on_read_bytes, (void *)on_done, call_id);
        free(req);

        if (err_id != 0) {
            fprintf(stderr, "ServerStreamCall failed: %d\n", err_id);
            return 1;
        }
        if (!st.done || st.done_error_id != 0) {
            fprintf(stderr, "expected done with error=0, got done=%d err=%" PRIu64 "\n", st.done, st.done_error_id);
            return 1;
        }
        if (st.count != 3) {
            fprintf(stderr, "expected 3 responses, got %d\n", st.count);
            return 1;
        }
        if (strcmp(st.results[0], "test-a") != 0 || strcmp(st.results[1], "test-b") != 0 || strcmp(st.results[2], "test-c") != 0) {
            fprintf(stderr, "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
            return 1;
        }
    }

    // Native server-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        int err_id = Ygrpc_StreamService_ServerStreamCall_Native(
            (char *)"test", 4, (int32_t)7,
            (void *)on_read_native,
            (void *)on_done,
            (uint64_t)(uintptr_t)&st);

        if (err_id != 0) {
            fprintf(stderr, "ServerStreamCall_Native failed: %d\n", err_id);
            return 1;
        }
        if (!st.done || st.done_error_id != 0) {
            fprintf(stderr, "expected done with error=0, got done=%d err=%" PRIu64 "\n", st.done, st.done_error_id);
            return 1;
        }
        if (st.count != 3) {
            fprintf(stderr, "expected 3 responses, got %d\n", st.count);
            return 1;
        }
        if (strcmp(st.results[0], "test-a") != 0 || strcmp(st.results[1], "test-b") != 0 || strcmp(st.results[2], "test-c") != 0) {
            fprintf(stderr, "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
            return 1;
        }
    }

    printf("server_stream_test OK\n");
    return 0;
}
