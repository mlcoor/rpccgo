# Tasks: Add global RPC dispatch registry

## 1. Specs
- [x] Add new capability spec delta: `rpc-dispatch`.
- [x] Validate with `openspec validate add-global-rpc-dispatch-registry --strict`.

## 2. Runtime (rpcruntime)
- [x] Add global registry keyed by `(protocol, serviceName)`.
- [x] Add global registration API (`RegisterHandler(serviceName, protocol, handler, ...)`) with replace semantics.
- [x] Add lookup API (`LookupHandler(serviceName, protocol)`).
- [x] Ensure thread-safety and low-overhead read path.

## 3. Codegen (adaptor)
（Not in this change）

本 change 仅交付运行时注册中心能力；adaptor 生成与路由调用将通过后续单独 change 提案完成。

## 4. Tests & Validation
- [x] Add focused unit tests for register/lookup and duplicate/replace handling.
- [x] Add concurrency sanity test (parallel invokes) if lightweight.
- [x] Run `go test ./...`.
