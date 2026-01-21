# Tasks: Add RPC CGO adaptor code generator

## 1. Specs
- [x] Add new capability spec delta: `rpc-cgo-adaptor`.
- [x] Add spec delta for `rpc-dispatch` to standardize protocol constants and usage.
- [x] Validate with `openspec validate add-rpc-cgo-adaptor-codegen --strict`.

## 2. Codegen: `protoc-gen-rpc-cgo-adaptor`
- [x] Add minimal protoc plugin harness using `google.golang.org/protobuf/compiler/protogen`.
- [x] Add protoc plugin options to control generated framework: `grpc` / `connectrpc` (default: connectrpc).
- [x] For each `service` in input protos, generate adaptor Go code that:
  - [x] Defines `serviceName` and `fullMethod` constants (`/Service/Method`).
  - [x] Defines one exported Go function per RPC method for CGO-side Go callers.
  - [x] Uses Go proto message structs in signatures.
  - [x] For unary methods, accepts `(ctx context.Context, req *pb.Req)` and returns `(*pb.Resp, error)`.
  - [x] For client-streaming methods, generates a staged API: `Start` / `Send(req *pb.Req)` / `Finish() (*pb.Resp, error)`.
  - [x] For server-streaming methods, generates a callback-based API that accepts `onRead` and `onDone`.
  - [x] For bidi-streaming methods, generates a combined staged + callback API: `Start` + `Send`/`CloseSend` + receive callbacks.
  - [x] Implement process-local `uint64` stream handles with deterministic lifecycle (invalid/finished handle returns error).
  - [x] Ensure `onRead` returning false terminates the underlying stream promptly (cancel context or equivalent), and `onDone` is invoked exactly once.
  - [x] Reads `protocol` from `ctx` (using runtime helpers) and looks up handler via `rpcruntime.LookupGrpcHandler` / `rpcruntime.LookupConnectHandler`.
  - [x] Performs type assertions to the expected service interface and invokes the method.
  - [x] Returns deterministic errors for: unknown protocol, unregistered service, handler type mismatch.

## 3. Tests & Validation
- [x] Add a dedicated streaming proto fixture under `test/proto/` (new file) covering client/server/bidi streaming, and regenerate both grpc and connect outputs to inspect real framework method shapes.
- [x] Run `go test ./...`.
- [x] Add integration tests for streaming methods (client-streaming, server-streaming, bidi-streaming).
- [x] Update `build-grpc.sh` and `build-connectrpc.sh` to include adaptor generation.

## 4. Documentation
- [x] Update `README.md` describing how adaptor functions are used by CGO-side Go code and how to pass `protocol`.
