#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

./run-adaptor-tests.sh all

echo ""
echo "=== Running C end-to-end tests ==="

./run-c-tests.sh all

echo ""
echo "=== All C tests completed ==="
