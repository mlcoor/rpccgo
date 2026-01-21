#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

./build-connect-suffix.sh

echo "Installing protoc-gen-rpc-cgo (from workspace)..."
(cd .. && go install ./cmd/protoc-gen-rpc-cgo)

mkdir -p ./cgo_connect_suffix
# Clean generated files but keep hand-written registry.
find ./cgo_connect_suffix -mindepth 1 -maxdepth 1 -type f ! -name 'registry.go' -delete
find ./cgo_connect_suffix -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix"

protoc -Iproto -I../proto \
  --rpc-cgo_out=./cgo_connect_suffix \
  --rpc-cgo_opt=paths=source_relative,${GO_PKG} \
  ./proto/unary.proto ./proto/stream.proto

echo "Building c-shared library (connectrpc suffix)..."
go build -buildmode=c-shared -o ./c_tests/libygrpc.so ./cgo_connect_suffix
cp ./cgo_connect_suffix/ygrpc_cgo_common.h ./c_tests/
