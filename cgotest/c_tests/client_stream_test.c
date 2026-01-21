#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libygrpc.h"
#include "proto_helpers.h"

static int g_free_called = 0;
static void counting_free(void* p) {
    g_free_called++;
    free(p);
}

static void expect_eq_str(const char* got, int got_len, const char* want) {
    int want_len = (int)strlen(want);
    if (got_len != want_len || memcmp(got, want, (size_t)want_len) != 0) {
        fprintf(stderr, "expected '%s' (%d), got '%.*s' (%d)\n", want, want_len, got_len, got, got_len);
        abort();
    }
}

int main(void) {
    int rc = Ygrpc_SetProtocol(YGRPC_PROTOCOL_UNSET);
    if (rc != 0)
    {
        fprintf(stderr, "Ygrpc_SetProtocol failed: %d\n", rc);
        return 1;
    }

    // Binary client-streaming: Start/Send/Finish
    {
        uint64_t handle = 0;
        int err_id = Ygrpc_StreamService_ClientStreamCallStart(&handle);
        if (err_id != 0 || handle == 0) {
            fprintf(stderr, "Start failed: err=%d handle=%llu\n", err_id, (unsigned long long)handle);
            return 1;
        }

        const char* msgs[] = {"A", "B", "C"};
        for (int i = 0; i < 3; i++) {
            uint8_t* req = NULL;
            int req_len = 0;
            assert(ygrpc_encode_stream_request(msgs[i], 1, i, &req, &req_len) == 0);
            err_id = Ygrpc_StreamService_ClientStreamCallSend(handle, req, req_len);
            free(req);
            if (err_id != 0) {
                fprintf(stderr, "Send failed: %d\n", err_id);
                return 1;
            }
        }

        void* resp_ptr = NULL;
        int resp_len = 0;
        FreeFunc resp_free = NULL;
        err_id = Ygrpc_StreamService_ClientStreamCallFinish(handle, &resp_ptr, &resp_len, &resp_free);
        if (err_id != 0) {
            fprintf(stderr, "Finish failed: %d\n", err_id);
            return 1;
        }

        const uint8_t* result_ptr = NULL;
        int result_len = 0;
        assert(ygrpc_decode_string_field((const uint8_t*)resp_ptr, resp_len, 1, &result_ptr, &result_len) == 0);
        expect_eq_str((const char*)result_ptr, result_len, "received:ABC");
        resp_free(resp_ptr);
    }

    // Native client-streaming
    {
        uint64_t handle = 0;
        int err_id = Ygrpc_StreamService_ClientStreamCallStart_Native(&handle);
        if (err_id != 0 || handle == 0) {
            fprintf(stderr, "Start_Native failed: err=%d handle=%llu\n", err_id, (unsigned long long)handle);
            return 1;
        }

        const char* msgs[] = {"A", "B", "C"};
        for (int i = 0; i < 3; i++) {
            err_id = Ygrpc_StreamService_ClientStreamCallSend_Native(handle, (char*)msgs[i], 1, (int32_t)i);
            if (err_id != 0) {
                fprintf(stderr, "Send_Native failed: %d\n", err_id);
                return 1;
            }
        }

        char* out_result = NULL;
        int out_result_len = 0;
        FreeFunc out_result_free = NULL;
        int32_t out_seq = 0;

        err_id = Ygrpc_StreamService_ClientStreamCallFinish_Native(handle, &out_result, &out_result_len, &out_result_free, &out_seq);
        if (err_id != 0) {
            fprintf(stderr, "Finish_Native failed: %d\n", err_id);
            return 1;
        }

        expect_eq_str(out_result, out_result_len, "received:ABC");
        if (out_result_free) out_result_free(out_result);
    }

    // Native client-streaming (TakeReq for string field)
    {
        uint64_t handle = 0;
        int err_id = Ygrpc_StreamService_ClientStreamCallStart_Native(&handle);
        if (err_id != 0 || handle == 0) {
            fprintf(stderr, "Start_Native failed: err=%d handle=%llu\n", err_id, (unsigned long long)handle);
            return 1;
        }

        g_free_called = 0;
        for (int i = 0; i < 2; i++) {
            char* p = (char*)malloc(1);
            p[0] = (i == 0) ? 'X' : 'Y';
            err_id = Ygrpc_StreamService_ClientStreamCallSend_Native_TakeReq(handle, p, 1, (FreeFunc)counting_free, (int32_t)i);
            if (err_id != 0) {
                fprintf(stderr, "Send_Native_TakeReq failed: %d\n", err_id);
                return 1;
            }
        }
        if (g_free_called != 2) {
            fprintf(stderr, "expected TakeReq frees=2, got %d\n", g_free_called);
            return 1;
        }

        char* out_result = NULL;
        int out_result_len = 0;
        FreeFunc out_result_free = NULL;
        int32_t out_seq = 0;
        err_id = Ygrpc_StreamService_ClientStreamCallFinish_Native(handle, &out_result, &out_result_len, &out_result_free, &out_seq);
        if (err_id != 0) {
            fprintf(stderr, "Finish_Native failed: %d\n", err_id);
            return 1;
        }

        // Handler concatenates; expect received:XY
        expect_eq_str(out_result, out_result_len, "received:XY");
        if (out_result_free) out_result_free(out_result);
    }

    printf("client_stream_test OK\n");
    return 0;
}
