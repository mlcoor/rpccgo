#!/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

if ! command -v protoc >/dev/null 2>&1; then
    echo "protoc is not installed. Please install Protocol Buffers compiler."
    exit 1
fi

# --- mix (grpc + connectrpc) ---
mkdir -p ./mix
# Clean generated files but keep hand-written tests.
find ./mix -mindepth 1 -maxdepth 1 -type f ! -name 'adaptor_test.go' -delete
find ./mix -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+'
(cd .. && go install ./cmd/protoc-gen-rpc-cgo-adaptor)

GO_PKG_mix="Munary.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix"
ADAPTOR_PROTOCOL="grpc|connectrpc"

protoc -Iproto -I../proto \
  --go_out=./mix --go_opt=paths=source_relative,${GO_PKG_mix} \
  --go-grpc_out=./mix --go-grpc_opt=paths=source_relative,${GO_PKG_mix} \
  --connect-go_out=./mix --connect-go_opt=paths=source_relative,simple=true,${GO_PKG_mix} \
  --rpc-cgo-adaptor_out=./mix \
  --rpc-cgo-adaptor_opt=paths=source_relative,protocol=${ADAPTOR_PROTOCOL},connect_package_suffix=connect,${GO_PKG_mix} \
  ./proto/unary.proto ./proto/stream.proto
