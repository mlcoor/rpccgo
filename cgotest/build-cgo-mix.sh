#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

./build-mix.sh

echo "Installing protoc-gen-rpc-cgo (from workspace)..."
(cd .. && go install ./cmd/protoc-gen-rpc-cgo)

mkdir -p ./cgo_mix
# Clean generated files but keep hand-written registry.
find ./cgo_mix -mindepth 1 -maxdepth 1 -type f ! -name 'registry.go' -delete
find ./cgo_mix -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'

GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix"

protoc -Iproto -I../proto \
  --rpc-cgo_out=./cgo_mix \
  --rpc-cgo_opt=paths=source_relative,${GO_PKG} \
  ./proto/unary.proto ./proto/stream.proto

echo "Building c-shared library (mix)..."
go build -buildmode=c-shared -o ./c_tests/libygrpc.so ./cgo_mix
cp ./cgo_mix/ygrpc_cgo_common.h ./c_tests/
