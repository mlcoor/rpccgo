#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

cfg="${1:-all}"

if [[ -z "${cfg}" ]]; then
	echo "usage: $0 <connect|grpc|mix|connect_suffix|all>"
	exit 2
fi

build_one() {
	local name="$1"
	echo ""
	echo ">>> Building ${name}..."
	"./build-${name}.sh"
}

test_one() {
	local dir="$1"
	local label="$2"
	echo ""
	echo ">>> Testing ${label}..."
	(cd "${dir}" && go test -v ./)
}

run_all() {
	echo "=== Building test packages ==="
	build_one connect
	build_one grpc
	build_one connect-suffix
	build_one mix

	echo ""
	echo "=== Running all adaptor tests ==="
	test_one connect "ConnectRPC"
	test_one grpc "gRPC"
	test_one mix "mix (grpc+connectrpc)"
	test_one connect_suffix "connect_suffix (connectrpc)"
	echo ""
	echo "=== All adaptor tests completed ==="
}

run_one() {
	local which="$1"
	case "${which}" in
		connect)
			echo "=== Building test package ==="
			build_one connect
			echo ""
			echo "=== Running adaptor tests ==="
			test_one connect "ConnectRPC"
			;;
		grpc)
			echo "=== Building test package ==="
			build_one grpc
			echo ""
			echo "=== Running adaptor tests ==="
			test_one grpc "gRPC"
			;;
		mix)
			echo "=== Building test package ==="
			build_one mix
			echo ""
			echo "=== Running adaptor tests ==="
			test_one mix "mix (grpc+connectrpc)"
			;;
		connect_suffix)
			echo "=== Building test package ==="
			build_one connect-suffix
			echo ""
			echo "=== Running adaptor tests ==="
			test_one connect_suffix "connect_suffix (connectrpc)"
			;;
		all)
			run_all
			;;
		*)
			echo "unknown cfg: ${which}"
			echo "usage: $0 <connect|grpc|mix|connect_suffix|all>"
			exit 2
			;;
	esac
}

run_one "${cfg}"
