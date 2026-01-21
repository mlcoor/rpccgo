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

echo "Installing protoc-gen-rpc-cgo-adaptor (from workspace)..."
(cd .. && go install ./cmd/protoc-gen-rpc-cgo-adaptor)

mkdir -p ./grpc
# Clean generated files but keep hand-written tests.
find ./grpc -mindepth 1 -maxdepth 1 -type f ! -name 'adaptor_test.go' -delete
find ./grpc -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

# Package mapping for cgotest_grpc
GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc"

protoc  -Iproto -I../proto --go_out=./grpc --go_opt=paths=source_relative,${GO_PKG} \
 --go-grpc_out=./grpc --go-grpc_opt=paths=source_relative,${GO_PKG} \
 ./proto/unary.proto ./proto/stream.proto

protoc -Iproto -I../proto --rpc-cgo-adaptor_out=./grpc \
 --rpc-cgo-adaptor_opt=paths=source_relative,protocol=grpc,${GO_PKG} \
 ./proto/unary.proto ./proto/stream.proto