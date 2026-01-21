#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

cfg="${1:-}"
if [[ -z "${cfg}" ]]; then
  echo "usage: $0 <connect|grpc|connect_suffix|mix|all>"
  exit 2
fi

cc_bin="${CC:-cc}"

cflags=(
  -O2
  -std=c11
  -Wall
  -Wextra
  -D_POSIX_C_SOURCE=200809L
  -I./c_tests
)

ldflags=(
  -L./c_tests
  -lygrpc
  -Wl,-rpath,'$ORIGIN'
)

build_one() {
  local src="$1"
  local out="$2"
  "${cc_bin}" "${cflags[@]}" "${src}" -o "./c_tests/${out}" "${ldflags[@]}"
}

build_and_run_tests() {
  local which="$1"
  echo "Building C tests (${which})..."
  build_one ./c_tests/unary_test.c unary_test
  build_one ./c_tests/client_stream_test.c client_stream_test
  build_one ./c_tests/server_stream_test.c server_stream_test
  build_one ./c_tests/bidi_stream_test.c bidi_stream_test

  echo "Running C tests (${which})..."
  (
    cd ./c_tests
    ./unary_test
    ./client_stream_test
    ./server_stream_test
    ./bidi_stream_test
  )

  echo "C tests OK (${which})"
}

run_one() {
  local which="$1"
  case "${which}" in
    connect)
      ./build-cgo-connect.sh
      ;;
    grpc)
      ./build-cgo-grpc.sh
      ;;
    connect_suffix)
      ./build-cgo-connect-suffix.sh
      ;;
    mix)
      ./build-cgo-mix.sh
      ;;
    *)
      echo "unknown cfg: ${which}"
      echo "usage: $0 <connect|grpc|connect_suffix|mix|all>"
      exit 2
      ;;
  esac

  build_and_run_tests "${which}"
}

run_all() {
  run_one connect
  echo ""
  run_one grpc
  echo ""
  run_one mix
  echo ""
  run_one connect_suffix
}

case "${cfg}" in
  all)
    run_all
    ;;
  *)
    run_one "${cfg}"
    ;;
esac
