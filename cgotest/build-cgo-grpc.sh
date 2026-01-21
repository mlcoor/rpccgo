#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

./build-grpc.sh

echo "Installing protoc-gen-rpc-cgo (from workspace)..."
(cd .. && go install ./cmd/protoc-gen-rpc-cgo)

mkdir -p ./cgo_grpc
# Clean generated files but keep hand-written registry.
find ./cgo_grpc -mindepth 1 -maxdepth 1 -type f ! -name 'registry.go' -delete
find ./cgo_grpc -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc"

protoc -Iproto -I../proto \
  --rpc-cgo_out=./cgo_grpc \
  --rpc-cgo_opt=paths=source_relative,${GO_PKG} \
  ./proto/unary.proto ./proto/stream.proto

echo "Building c-shared library (grpc)..."
go build -buildmode=c-shared -o ./c_tests/libygrpc.so ./cgo_grpc
cp ./cgo_grpc/ygrpc_cgo_common.h ./c_tests/
