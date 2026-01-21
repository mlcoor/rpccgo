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

# --- connect_suffix (connectrpc only, connect-go in suffix package) ---
mkdir -p ./connect_suffix
# Clean generated files but keep hand-written tests.
find ./connect_suffix -mindepth 1 -maxdepth 1 -type f ! -name 'adaptor_test.go' -delete
find ./connect_suffix -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

GO_PKG_CONNECT_SUFFIX="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix"

protoc -Iproto -I../proto \
  --go_out=./connect_suffix --go_opt=paths=source_relative,${GO_PKG_CONNECT_SUFFIX} \
  --connect-go_out=./connect_suffix --connect-go_opt=paths=source_relative,simple=true,${GO_PKG_CONNECT_SUFFIX} \
  --rpc-cgo-adaptor_out=./connect_suffix \
  --rpc-cgo-adaptor_opt=paths=source_relative,protocol=connectrpc,connect_package_suffix=connect,${GO_PKG_CONNECT_SUFFIX} \
    ./proto/unary.proto ./proto/stream.proto
