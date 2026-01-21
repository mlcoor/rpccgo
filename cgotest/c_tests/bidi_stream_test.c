#include <assert.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "libygrpc.h"
#include "proto_helpers.h"

typedef struct {
    int done;
    int done_error_id;
    int count;
    char results[16][64];
} stream_state;

static stream_state *g_state = NULL;
static uint64_t g_call_id = 0;

static stream_state *get_state(uint64_t call_id)
{
    if (!g_state)
    {
        fprintf(stderr, "callback before state init\n");
        abort();
    }
    if (g_call_id == 0)
    {
        g_call_id = call_id;
    }
    else if (g_call_id != call_id)
    {
        fprintf(stderr, "unexpected call_id: got=%llu want=%llu\n",
                (unsigned long long)call_id, (unsigned long long)g_call_id);
        abort();
    }
    return g_state;
}

static void on_read_bytes(uint64_t call_id, void *resp_ptr, int resp_len, FreeFunc resp_free)
{
    stream_state *st = get_state(call_id);

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

static void on_done(uint64_t call_id, int error_id)
{
    stream_state *st = get_state(call_id);
    st->done = 1;
    st->done_error_id = error_id;
}

static void on_read_native(uint64_t call_id, void *result_ptr, int result_len, FreeFunc result_free, int32_t sequence)
{
    (void)sequence;
    stream_state *st = get_state(call_id);
    int n = result_len;
    if (n > (int)sizeof(st->results[0]) - 1) n = (int)sizeof(st->results[0]) - 1;
    memcpy(st->results[st->count], result_ptr, (size_t)n);
    st->results[st->count][n] = 0;
    st->count++;
    if (result_free) result_free(result_ptr);
}

static void wait_done(stream_state* st) {
    for (int i = 0; i < 200; i++) {
        if (st->done) return;
        struct timespec ts;
        ts.tv_sec = 0;
        ts.tv_nsec = 10 * 1000 * 1000; // 10ms
        nanosleep(&ts, NULL);
    }
}

int main(void) {
    int rc = Ygrpc_SetProtocol(YGRPC_PROTOCOL_UNSET);
    if (rc != 0)
    {
        fprintf(stderr, "Ygrpc_SetProtocol failed: %d\n", rc);
        return 1;
    }

    // Binary bidi-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        g_state = &st;
        g_call_id = 0;

        uint64_t handle = 0;
        int err_id = Ygrpc_StreamService_BidiStreamCallStart((void *)on_read_bytes, (void *)on_done, &handle);
        if (err_id != 0 || handle == 0) {
            fprintf(stderr, "BidiStart failed: err=%d handle=%llu\n", err_id, (unsigned long long)handle);
            return 1;
        }
        if (g_call_id == 0)
            g_call_id = handle;

        const char* msgs[] = {"X", "Y", "Z"};
        for (int i = 0; i < 3; i++) {
            uint8_t* req = NULL;
            int req_len = 0;
            assert(ygrpc_encode_stream_request(msgs[i], 1, i, &req, &req_len) == 0);
            err_id = Ygrpc_StreamService_BidiStreamCallSend(handle, req, req_len);
            free(req);
            if (err_id != 0) {
                fprintf(stderr, "BidiSend failed: %d\n", err_id);
                return 1;
            }
        }

        err_id = Ygrpc_StreamService_BidiStreamCallCloseSend(handle);
        if (err_id != 0) {
            fprintf(stderr, "BidiCloseSend failed: %d\n", err_id);
            return 1;
        }

        wait_done(&st);
        if (!st.done || st.done_error_id != 0) {
            fprintf(stderr, "expected done with error=0, got done=%d err=%d\n", st.done, st.done_error_id);
            return 1;
        }
        if (st.count != 3) {
            fprintf(stderr, "expected 3 responses, got %d\n", st.count);
            return 1;
        }
        if (strcmp(st.results[0], "echo:X") != 0 || strcmp(st.results[1], "echo:Y") != 0 || strcmp(st.results[2], "echo:Z") != 0) {
            fprintf(stderr, "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
            return 1;
        }
    }

    // Native bidi-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        g_state = &st;
        g_call_id = 0;

        uint64_t handle = 0;
        int err_id = Ygrpc_StreamService_BidiStreamCallStart_Native((void *)on_read_native, (void *)on_done, &handle);
        if (err_id != 0 || handle == 0) {
            fprintf(stderr, "BidiStart_Native failed: err=%d handle=%llu\n", err_id, (unsigned long long)handle);
            return 1;
        }
        if (g_call_id == 0)
            g_call_id = handle;

        const char* msgs[] = {"X", "Y", "Z"};
        for (int i = 0; i < 3; i++) {
            err_id = Ygrpc_StreamService_BidiStreamCallSend_Native(handle, (char*)msgs[i], 1, (int32_t)i);
            if (err_id != 0) {
                fprintf(stderr, "BidiSend_Native failed: %d\n", err_id);
                return 1;
            }
        }

        err_id = Ygrpc_StreamService_BidiStreamCallCloseSend_Native(handle);
        if (err_id != 0) {
            fprintf(stderr, "BidiCloseSend_Native failed: %d\n", err_id);
            return 1;
        }

        wait_done(&st);
        if (!st.done || st.done_error_id != 0) {
            fprintf(stderr, "expected done with error=0, got done=%d err=%d\n", st.done, st.done_error_id);
            return 1;
        }
        if (st.count != 3) {
            fprintf(stderr, "expected 3 responses, got %d\n", st.count);
            return 1;
        }
        if (strcmp(st.results[0], "echo:X") != 0 || strcmp(st.results[1], "echo:Y") != 0 || strcmp(st.results[2], "echo:Z") != 0) {
            fprintf(stderr, "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
            return 1;
        }
    }

    printf("bidi_stream_test OK\n");
    return 0;
}
