# Tasks: Add `protoc-gen-rpc-cgo` (CGO C ABI codegen)

## 1. Spec + Proposal
- [ ] Validate proposal docs: `openspec validate add-protoc-gen-rpc-cgo-plugin --strict`

## 2. Plugin: `cmd/protoc-gen-rpc-cgo/`
- [ ] Implement proto options parsing for request free strategy and native:
  - [ ] `ygrpc_cgo_req_free_default` / `ygrpc_cgo_req_free_method`
  - [ ] `ygrpc_cgo_native_default` / `ygrpc_cgo_native`
  - [ ] method option overrides file option; absent uses defaults.
- [ ] Generate `main.go` (package main) and one `<fileName>_cgo.go` per proto input file (prefix = GeneratedFilenamePrefix).
- [ ] Generate C preamble with include-guards for `FreeFunc` and `Ygrpc_GetErrorMsg` declaration.
- [ ] Implement Binary ABI generation:
  - [ ] Unary
  - [ ] Client-streaming (Start/Send/Finish)
  - [ ] Server-streaming (callback)
  - [ ] Bidi-streaming (Start registers callbacks; Send; CloseSend)
- [ ] Implement TakeReq variants per resolved `free_strategy`.
- [ ] Implement Native ABI generation gated by flat-message eligibility, including `_Native_TakeReq` combinations per resolved `native` option.
- [ ] Ensure all exported functions:
  - [ ] Return `0` on success
  - [ ] On error: `rpcruntime.StoreError(err)` and return error id

## 3. Integration: adaptor linkage
- [ ] Ensure generated CGO code imports and calls the generated adaptor API (from `protoc-gen-rpc-cgo-adaptor`).
- [ ] Confirm protocol selection semantics match `rpc-cgo-adaptor` spec (context protocol).

## 4. Tests: `cgotest/` end-to-end
- [ ] Add C test harness programs covering:
  - [ ] Unary (Binary + Native)
  - [ ] Client-streaming (Binary + Native)
  - [ ] Server-streaming (Binary + Native)
  - [ ] Bidi-streaming (Binary + Native)
- [ ] For Binary tests, implement protobuf bytes serialize/deserialize in C to build requests and validate responses.
- [ ] Add matrix runners for adaptor configs: `grpc`, `connect`, `connect_suffix`, `mix`.
- [ ] Ensure tests validate error path:
  - [ ] Non-zero error id returned
  - [ ] `Ygrpc_GetErrorMsg` returns message and free func works

## 5. Docs + Scripts
- [ ] Update `cgotest/README.md` with exact commands.
- [ ] Add/extend build scripts (e.g. `cgotest/build-cgo-*.sh`) to:
  - [ ] run `protoc`
  - [ ] build `.so`
  - [ ] compile/run C tests

## 6. Validation
- [ ] `go test ./...`
- [ ] `cgotest/test.sh` (or equivalent runner added in this change)
