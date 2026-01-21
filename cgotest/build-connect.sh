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

echo "Installing protoc-gen-rpc-cgo-adaptor (from workspace)..."
(cd .. && go install ./cmd/protoc-gen-rpc-cgo-adaptor)


mkdir -p ./connect
# Clean generated files but keep hand-written tests.
find ./connect -mindepth 1 -maxdepth 1 -type f ! -name 'adaptor_test.go' -delete
find ./connect -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

# Package mapping for cgotest_connect
GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect"

# 生成 Connect 代码，放在 ./connect 目录下
protoc  -Iproto -I../proto --go_out=./connect --go_opt=paths=source_relative,${GO_PKG} \
 --connect-go_out=./connect --connect-go_opt=package_suffix="",paths=source_relative,simple=true,${GO_PKG} \
 ./proto/unary.proto ./proto/stream.proto

# 生成 CGO adaptor 代码
protoc -Iproto -I../proto --rpc-cgo-adaptor_out=./connect \
 --rpc-cgo-adaptor_opt=paths=source_relative,protocol=connectrpc,${GO_PKG} \
 ./proto/unary.proto ./proto/stream.proto
