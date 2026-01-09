# Tasks: Add runtime error registry and `Ygrpc_GetErrorMsg` ABI

## 1. Proposal deliverables
- [ ] Ensure each requirement includes at least one scenario.
- [ ] Run `openspec validate add-rpc-runtime-error-registry --strict` and fix all issues.

## 2. Apply-stage implementation checklist (future work)
- [ ] Add `rpcruntime/errors.go`: errorId allocation + message storage.
- [ ] Add `rpcruntime/errors_ttl.go`: TTL eviction/cleanup (~3 seconds).
- [ ] Add `rpcruntime/errmsg.go`: helper for `Ygrpc_GetErrorMsg` wrapper to retrieve message bytes safely.
- [ ] Add `rpcruntime/errors_test.go`: covers store/retrieve, not-found after TTL, and concurrency safety.
- [ ] Add a minimal build check (local): `go test ./...`.
