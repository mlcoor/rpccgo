#!/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

if ! command -v protoc >/dev/null 2>&1; then
    echo "protoc is not installed. Please install Protocol Buffers compiler."
    exit 1
fi

if ! command -v protoc-gen-go >/dev/null 2>&1; then
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-connect-go >/dev/null 2>&1; then
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.0
fi

if ! protoc-gen-connect-go --version | grep -q "v1.19.0"; then
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.0
fi


mkdir -p ./connect

# 生成 Connect 代码，放在 ./connect 目录下
protoc  -Iproto --go_out=./connect --go_opt=paths=source_relative \
 --connect-go_out=./connect --connect-go_opt=package_suffix="",paths=source_relative,simple \
 ./proto/test.proto ./proto/test2.proto
