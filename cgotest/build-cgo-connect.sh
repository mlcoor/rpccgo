#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

./build-connect.sh

echo "Installing protoc-gen-rpc-cgo (from workspace)..."
(cd .. && go install ./cmd/protoc-gen-rpc-cgo)

mkdir -p ./cgo_connect
# Clean generated files but keep hand-written registry.
find ./cgo_connect -mindepth 1 -maxdepth 1 -type f ! -name 'registry.go' -delete
find ./cgo_connect -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect"

protoc -Iproto -I../proto \
  --rpc-cgo_out=./cgo_connect \
  --rpc-cgo_opt=paths=source_relative,${GO_PKG} \
  ./proto/unary.proto ./proto/stream.proto

echo "Building c-shared library (connectrpc)..."
go build -buildmode=c-shared -o ./c_tests/libygrpc.so ./cgo_connect
cp ./cgo_connect/ygrpc_cgo_common.h ./c_tests/
