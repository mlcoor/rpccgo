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

static void test_error_path(void) {
    // Feed invalid protobuf bytes to force an error path.
    uint8_t bad[1] = {0xFF};

    void* resp_ptr = NULL;
    int resp_len = 0;
    FreeFunc resp_free = NULL;

    int err_id = Ygrpc_TestService_Ping(bad, 1, &resp_ptr, &resp_len, &resp_free);
    if (err_id == 0) {
        fprintf(stderr, "expected non-zero error id in invalid-protobuf test\n");
        abort();
    }

    void* emsg_ptr = NULL;
    int emsg_len = 0;
    FreeFunc emsg_free = NULL;
    int rc = Ygrpc_GetErrorMsg(err_id, &emsg_ptr, &emsg_len, &emsg_free);
    if (rc != 0 || emsg_len <= 0 || emsg_ptr == NULL || emsg_free == NULL) {
        fprintf(stderr, "Ygrpc_GetErrorMsg failed: rc=%d len=%d ptr=%p free=%p\n", rc, emsg_len, emsg_ptr, (void*)emsg_free);
        abort();
    }

    // Just ensure free works.
    emsg_free(emsg_ptr);
}

int main(void) {
    int rc = Ygrpc_SetProtocol(YGRPC_PROTOCOL_UNSET);
    if (rc != 0)
    {
        fprintf(stderr, "Ygrpc_SetProtocol failed: %d\n", rc);
        return 1;
    }

    // Binary unary
    {
        const char* msg = "hello";
        uint8_t* req_buf = NULL;
        int req_len = 0;
        assert(ygrpc_encode_string_field(1, (const uint8_t*)msg, (int)strlen(msg), &req_buf, &req_len) == 0);

        void* resp_ptr = NULL;
        int resp_len = 0;
        FreeFunc resp_free = NULL;

        int err_id = Ygrpc_TestService_Ping(req_buf, req_len, &resp_ptr, &resp_len, &resp_free);
        free(req_buf);

        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping failed: %d\n", err_id);
            return 1;
        }
        if (resp_ptr == NULL || resp_free == NULL) {
            fprintf(stderr, "expected resp_ptr and resp_free\n");
            return 1;
        }

        const uint8_t* out_msg_ptr = NULL;
        int out_msg_len = 0;
        assert(ygrpc_decode_string_field((const uint8_t*)resp_ptr, resp_len, 1, &out_msg_ptr, &out_msg_len) == 0);
        expect_eq_str((const char*)out_msg_ptr, out_msg_len, "pong: hello");

        resp_free(resp_ptr);
    }

    // Binary unary (TakeReq)
    {
        g_free_called = 0;
        const char* msg = "hello";
        uint8_t* req_buf = NULL;
        int req_len = 0;
        assert(ygrpc_encode_string_field(1, (const uint8_t*)msg, (int)strlen(msg), &req_buf, &req_len) == 0);

        void* resp_ptr = NULL;
        int resp_len = 0;
        FreeFunc resp_free = NULL;

        int err_id = Ygrpc_TestService_Ping_TakeReq(req_buf, req_len, (FreeFunc)counting_free, &resp_ptr, &resp_len, &resp_free);
        // req_buf is freed by counting_free.

        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping_TakeReq failed: %d\n", err_id);
            return 1;
        }
        if (g_free_called != 1) {
            fprintf(stderr, "expected req free called once, got %d\n", g_free_called);
            return 1;
        }

        resp_free(resp_ptr);
    }

    // Native unary
    {
        const char* msg = "hello";
        char* out_msg = NULL;
        int out_len = 0;
        FreeFunc out_free = NULL;

        int err_id = Ygrpc_TestService_Ping_Native((char*)msg, (int)strlen(msg), &out_msg, &out_len, &out_free);
        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping_Native failed: %d\n", err_id);
            return 1;
        }
        if (out_msg == NULL || out_free == NULL) {
            fprintf(stderr, "expected native resp buffer/free\n");
            return 1;
        }
        expect_eq_str(out_msg, out_len, "pong: hello");
        out_free(out_msg);
    }

    test_error_path();

    printf("unary_test OK\n");
    return 0;
}
