#!/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

echo "=== Building test packages ==="

echo ""
echo ">>> Building ConnectRPC test package..."
./build-connectrpc.sh

echo ""
echo ">>> Building gRPC test package..."
./build-grpc.sh

echo ""
echo "=== Running all tests ==="

echo ""
echo ">>> Testing ConnectRPC..."
(cd connect && go test -v ./)

echo ""
echo ">>> Testing gRPC..."
(cd grpc && go test -v ./)

echo ""
echo "=== All tests completed ==="
