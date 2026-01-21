#pragma once

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static inline int ygrpc_varint_len_u32(uint32_t v) {
    int n = 1;
    while (v >= 0x80) {
        v >>= 7;
        n++;
    }
    return n;
}

static inline int ygrpc_put_varint_u32(uint8_t* out, uint32_t v) {
    int i = 0;
    while (v >= 0x80) {
        out[i++] = (uint8_t)((v & 0x7F) | 0x80);
        v >>= 7;
    }
    out[i++] = (uint8_t)(v & 0x7F);
    return i;
}

static inline int ygrpc_get_varint_u32(const uint8_t* in, int in_len, int* offset, uint32_t* out) {
    uint32_t v = 0;
    int shift = 0;
    int i = *offset;
    while (i < in_len && shift < 35) {
        uint8_t b = in[i++];
        v |= (uint32_t)(b & 0x7F) << shift;
        if ((b & 0x80) == 0) {
            *offset = i;
            *out = v;
            return 0;
        }
        shift += 7;
    }
    return -1;
}

static inline int ygrpc_encode_string_field(uint32_t field_no, const uint8_t* s, int s_len, uint8_t** out_buf, int* out_len) {
    uint32_t tag = (field_no << 3) | 2; // len-delimited
    int tag_len = ygrpc_varint_len_u32(tag);
    int len_len = ygrpc_varint_len_u32((uint32_t)s_len);
    int total = tag_len + len_len + s_len;

    uint8_t* buf = (uint8_t*)malloc((size_t)total);
    if (!buf) return -1;

    int off = 0;
    off += ygrpc_put_varint_u32(buf + off, tag);
    off += ygrpc_put_varint_u32(buf + off, (uint32_t)s_len);
    memcpy(buf + off, s, (size_t)s_len);

    *out_buf = buf;
    *out_len = total;
    return 0;
}

static inline int ygrpc_encode_stream_request(const char* data, int data_len, int32_t sequence, uint8_t** out_buf, int* out_len) {
    // StreamRequest { string data = 1; int32 sequence = 2; }
    uint32_t tag1 = (1u << 3) | 2u;
    uint32_t tag2 = (2u << 3) | 0u;

    int tag1_len = ygrpc_varint_len_u32(tag1);
    int data_len_len = ygrpc_varint_len_u32((uint32_t)data_len);
    int tag2_len = ygrpc_varint_len_u32(tag2);

    // int32 varint (sequence assumed non-negative in tests)
    uint32_t seq_u = (uint32_t)sequence;
    int seq_len = ygrpc_varint_len_u32(seq_u);

    int total = tag1_len + data_len_len + data_len + tag2_len + seq_len;
    uint8_t* buf = (uint8_t*)malloc((size_t)total);
    if (!buf) return -1;

    int off = 0;
    off += ygrpc_put_varint_u32(buf + off, tag1);
    off += ygrpc_put_varint_u32(buf + off, (uint32_t)data_len);
    memcpy(buf + off, data, (size_t)data_len);
    off += data_len;
    off += ygrpc_put_varint_u32(buf + off, tag2);
    off += ygrpc_put_varint_u32(buf + off, seq_u);

    *out_buf = buf;
    *out_len = total;
    return 0;
}

static inline int ygrpc_decode_string_field(const uint8_t* buf, int buf_len, uint32_t field_no, const uint8_t** out_ptr, int* out_len) {
    int off = 0;
    while (off < buf_len) {
        uint32_t tag = 0;
        if (ygrpc_get_varint_u32(buf, buf_len, &off, &tag) != 0) return -1;

        uint32_t wire = tag & 0x7u;
        uint32_t no = tag >> 3;

        if (wire == 2u) {
            uint32_t l = 0;
            if (ygrpc_get_varint_u32(buf, buf_len, &off, &l) != 0) return -1;
            if (off + (int)l > buf_len) return -1;
            if (no == field_no) {
                *out_ptr = buf + off;
                *out_len = (int)l;
                return 0;
            }
            off += (int)l;
            continue;
        }

        if (wire == 0u) {
            uint32_t v = 0;
            if (ygrpc_get_varint_u32(buf, buf_len, &off, &v) != 0) return -1;
            continue;
        }

        // Unsupported wire type for these tests.
        return -1;
    }

    return -1;
}

static inline int ygrpc_decode_int32_field(const uint8_t* buf, int buf_len, uint32_t field_no, int32_t* out) {
    int off = 0;
    while (off < buf_len) {
        uint32_t tag = 0;
        if (ygrpc_get_varint_u32(buf, buf_len, &off, &tag) != 0) return -1;

        uint32_t wire = tag & 0x7u;
        uint32_t no = tag >> 3;

        if (wire == 0u) {
            uint32_t v = 0;
            if (ygrpc_get_varint_u32(buf, buf_len, &off, &v) != 0) return -1;
            if (no == field_no) {
                *out = (int32_t)v;
                return 0;
            }
            continue;
        }

        if (wire == 2u) {
            uint32_t l = 0;
            if (ygrpc_get_varint_u32(buf, buf_len, &off, &l) != 0) return -1;
            off += (int)l;
            continue;
        }

        return -1;
    }

    return -1;
}
