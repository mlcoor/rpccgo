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

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

mkdir -p ./grpc

protoc  -Iproto --go_out=./grpc --go_opt=paths=source_relative \
 --go-grpc_out=./grpc --go-grpc_opt=paths=source_relative \
 ./proto/test.proto ./proto/test2.proto