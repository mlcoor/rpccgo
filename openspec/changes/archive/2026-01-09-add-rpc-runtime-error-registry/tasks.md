# Tasks: Add runtime error registry and `Ygrpc_GetErrorMsg` ABI

## 1. Proposal deliverables
- [x] Ensure each requirement includes at least one scenario.
- [x] Run `openspec validate add-rpc-runtime-error-registry --strict` and fix all issues.

## 2. Apply-stage implementation checklist (future work)
- [x] Add `rpcruntime/errors.go`: errorId allocation + message storage.
- [x] Add `rpcruntime/errors_ttl.go`: TTL eviction/cleanup (~3 seconds).
- [x] Add `rpcruntime/errmsg.go`: helper for `Ygrpc_GetErrorMsg` wrapper to retrieve message bytes safely.
- [x] Add `rpcruntime/errors_test.go`: covers store/retrieve, not-found after TTL, and concurrency safety.
- [x] Add `cabi/ygrpc_geterrmsg_cgo.go`: reference/template CGO-exported `Ygrpc_GetErrorMsg`.
- [x] Add documentation for rationale and usage (`README.md`, `rpcruntime/doc.go`, `cabi/README.md`).
- [x] Add a minimal build check (local): `go test ./...`.
