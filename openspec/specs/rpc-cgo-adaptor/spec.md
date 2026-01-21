# rpc-cgo-adaptor Specification

## Purpose
TBD - created by archiving change add-rpc-cgo-adaptor-codegen. Update Purpose after archive.
## Requirements
### Requirement: Generate Go-callable adaptor functions per service method
The system SHALL provide a protoc plugin `protoc-gen-rpc-cgo-adaptor`.
For each protobuf `service` and each RPC `method` (unary, client-streaming, server-streaming, bidi-streaming), the plugin SHALL generate Go adaptor entrypoints that can be called from CGO-side Go code.

The generated function:
- SHALL accept a `context.Context` parameter.
- For unary methods, SHALL accept request as the generated Go proto message struct type and SHALL return response as the generated Go proto message struct type and an `error`.
- For streaming methods, SHALL provide a type-safe API that exchanges proto message structs and does not require the caller to construct framework stream objects.

#### Scenario: Unary call signature uses proto message structs
- **GIVEN** a proto service `S` with a unary method `M(Req) returns (Resp)`
- **WHEN** `protoc-gen-rpc-cgo-adaptor` generates code
- **THEN** it SHALL generate a Go function whose request type is `*Req` and response type is `*Resp`
- **AND** it SHALL include `ctx context.Context` in parameters

#### Scenario: Streaming entrypoints exist and remain type-safe
- **GIVEN** a proto service `S` with a streaming method `M`
- **WHEN** `protoc-gen-rpc-cgo-adaptor` generates code
- **THEN** it SHALL generate adaptor entrypoints for `M` that exchange proto message structs
- **AND** it SHALL include `ctx context.Context` in parameters where applicable

---

### Requirement: Frameworks selection via plugin options
The protoc plugin SHALL support an option to control which framework-specific adaptor code is generated:
- `grpc`
- `connectrpc`

If no option is provided, the plugin SHALL generate adaptor code for `connectrpc` (default).

#### Scenario: Default generates connectrpc
- **GIVEN** a proto input with at least one service
- **WHEN** `protoc-gen-rpc-cgo-adaptor` runs with no framework selection option
- **THEN** it SHALL generate adaptor entrypoints for `connectrpc`

#### Scenario: Option restricts to grpc
- **GIVEN** a proto input with at least one service
- **WHEN** `protoc-gen-rpc-cgo-adaptor` runs with a framework option selecting `grpc` only
- **THEN** it SHALL generate adaptor entrypoints only for `grpc`

#### Scenario: Option restricts to connectrpc
- **GIVEN** a proto input with at least one service
- **WHEN** `protoc-gen-rpc-cgo-adaptor` runs with a framework option selecting `connectrpc` only
- **THEN** it SHALL generate adaptor entrypoints only for `connectrpc`

---

### Requirement: Client-streaming uses staged calls
For a client-streaming method, the generated adaptor API SHALL be split into stages:
- `Start`: initializes a stream session.
- `Send`: sends one request message (may be called multiple times).
- `Finish`: closes the send-side and returns the final response.

The staged API SHALL represent the in-flight stream using a process-local opaque handle of type `uint64`.

The staged API SHALL be type-safe (request/response use the Go proto message struct types).

#### Scenario: Client-streaming is Start/Send/Finish
- **GIVEN** a service `S` with a client-streaming method `M(stream Req) returns (Resp)`
- **WHEN** adaptor code is generated
- **THEN** it SHALL generate `Start`, `Send`, and `Finish` entrypoints for `M`
- **AND** `Send` SHALL accept `*Req`
- **AND** `Finish` SHALL return `*Resp` and `error`

---

### Requirement: Stream handle lifecycle is well-defined
The generated adaptor code SHALL ensure stream handles are safe and deterministic:
- `Start` SHALL return a non-zero `uint64` handle when successful.
- `Send` and `Finish` SHALL return a non-nil error when the provided handle is unknown, already finished, or otherwise invalid.
- After `Finish` returns (success or error), the handle SHALL become invalid and MUST NOT be reused.

#### Scenario: Invalid handle returns error
- **GIVEN** a `streamHandle` that was never returned by `Start` (or has already been finished)
- **WHEN** `Send(streamHandle, ...)` or `Finish(streamHandle)` is called
- **THEN** it SHALL return a non-nil error

---

### Requirement: Server-streaming uses callbacks
For a server-streaming method, the generated adaptor API SHALL accept callbacks:
- `onRead` invoked once per streamed response message.
- `onDone` invoked exactly once when the stream ends or fails.

The callback API SHALL be type-safe (response uses the Go proto message struct type).

If `onRead` returns `false`, the adaptor SHALL stop receiving further messages and SHALL promptly cancel/terminate the underlying stream to avoid resource leaks.

#### Scenario: Server-streaming invokes onRead and onDone
- **GIVEN** a service `S` with a server-streaming method `M(Req) returns (stream Resp)`
- **WHEN** adaptor code is generated
- **THEN** it SHALL generate an entrypoint for `M` that accepts `onRead` and `onDone`
- **AND** it SHALL invoke `onRead(*Resp)` for each response message
- **AND** it SHALL invoke `onDone(error)` once at completion

#### Scenario: onRead false stops and terminates stream
- **GIVEN** `onRead` returns `false` on the Nth message
- **WHEN** the server-streaming entrypoint is running
- **THEN** it SHALL stop reading further messages
- **AND** it SHALL terminate the underlying stream promptly
- **AND** it SHALL invoke `onDone(error)` once

---

### Requirement: Dispatch via global registry using protocol selection
At runtime, the generated adaptor function SHALL select the dispatch lookup path based on the `protocol` value carried in the provided `ctx`:
- For `rpcruntime.ProtocolGrpc`, it SHALL use `rpcruntime.LookupGrpcHandler(serviceName)`.
- For `rpcruntime.ProtocolConnectRPC`, it SHALL use `rpcruntime.LookupConnectHandler(serviceName)`.

The adaptor SHALL type-assert the returned handler to the expected service interface and invoke the concrete method.

#### Scenario: Grpc protocol dispatches to grpc handler
- **GIVEN** `ctx` carries `protocol = rpcruntime.ProtocolGrpc`
- **AND** a grpc handler is registered for `serviceName`
- **WHEN** the generated adaptor function is invoked
- **THEN** it SHALL lookup via `LookupGrpcHandler`
- **AND** call the grpc service method implementation

#### Scenario: Connectrpc protocol dispatches to connect handler
- **GIVEN** `ctx` carries `protocol = rpcruntime.ProtocolConnectRPC`
- **AND** a connectrpc handler is registered for `serviceName`
- **WHEN** the generated adaptor function is invoked
- **THEN** it SHALL lookup via `LookupConnectHandler`
- **AND** call the connectrpc service method implementation

---

### Requirement: Deterministic errors for routing failures
The generated adaptor code SHALL return deterministic errors for at least the following cases:
- Missing/unknown/unsupported `protocol` value in `ctx`
- Handler not registered for `(protocol, serviceName)`
- Registered handler has an unexpected type (type assertion fails)

#### Scenario: Unknown protocol returns error
- **GIVEN** `ctx` carries a `protocol` value that is not `ProtocolGrpc` or `ProtocolConnectRPC`
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

#### Scenario: Missing protocol returns error
- **GIVEN** `ctx` does not carry a protocol value
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

#### Scenario: Not registered returns error
- **GIVEN** no handler is registered for the selected `(protocol, serviceName)`
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

#### Scenario: Type mismatch returns error
- **GIVEN** a handler is registered for `(protocol, serviceName)`
- **AND** the handler cannot be asserted to the expected service interface
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

---

### Requirement: FullMethod constants for observability
The generated adaptor code SHALL define `fullMethod` constants in the form `/Service/Method` for each RPC method.

#### Scenario: FullMethod constant exists
- **GIVEN** a service `S` and method `M`
- **WHEN** adaptor code is generated
- **THEN** it SHALL define a constant equal to `/S/M`

