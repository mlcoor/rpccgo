# Tasks: Add `protoc-gen-rpc-cgo` (CGO C ABI codegen)

## 1. Spec + Proposal
- [x] Validate proposal docs: `openspec validate add-protoc-gen-rpc-cgo-plugin --strict`

## 2. Plugin: `cmd/protoc-gen-rpc-cgo/`
- [x] Implement proto options parsing for request free strategy and native:
  - [x] `ygrpc_cgo_req_free_default` / `ygrpc_cgo_req_free_method`
  - [x] `ygrpc_cgo_native_default` / `ygrpc_cgo_native`
  - [x] method option overrides file option; absent uses defaults.
- [x] Generate `main.go` (package main) and one `<fileName>_cgo.go` per proto input file (prefix = GeneratedFilenamePrefix).
- [x] Generate C preamble with include-guards for `FreeFunc` and `Ygrpc_GetErrorMsg` declaration.
- [x] Implement Binary ABI generation:
  - [x] Unary
  - [x] Client-streaming (Start/Send/Finish)
  - [x] Server-streaming (callback)
  - [x] Bidi-streaming (Start registers callbacks; Send; CloseSend)
- [x] Implement TakeReq variants per resolved `free_strategy`.
- [x] Implement Native ABI generation gated by flat-message eligibility, including `_Native_TakeReq` combinations per resolved `native` option.
- [x] Ensure all exported functions:
  - [x] Return `0` on success
  - [x] On error: `rpcruntime.StoreError(err)` and return error id

## 3. Integration: adaptor linkage
- [x] Ensure generated CGO code imports and calls the generated adaptor API (from `protoc-gen-rpc-cgo-adaptor`).
- [x] Confirm protocol selection semantics match `rpc-cgo-adaptor` spec (context protocol).

## 4. Tests: `cgotest/` end-to-end
- [x] Add C test harness programs covering:
  - [x] Unary (Binary + Native)
  - [x] Client-streaming (Binary + Native)
  - [x] Server-streaming (Binary + Native)
  - [x] Bidi-streaming (Binary + Native)
- [x] For Binary tests, implement protobuf bytes serialize/deserialize in C to build requests and validate responses.
- [x] Add matrix runners for adaptor configs: `grpc`, `connect`, `connect_suffix`, `mix`.
- [x] Ensure tests validate error path:
  - [x] Non-zero error id returned
  - [x] `Ygrpc_GetErrorMsg` returns message and free func works

## 5. Docs + Scripts
- [x] Update `cgotest/README.md` with exact commands.
- [x] Add/extend build scripts (e.g. `cgotest/build-cgo-*.sh`) to:
  - [x] run `protoc`
  - [x] build `.so`
  - [x] compile/run C tests

## 6. Validation
- [x] `go test ./...`
- [x] `cgotest/test.sh` (or equivalent runner added in this change)
