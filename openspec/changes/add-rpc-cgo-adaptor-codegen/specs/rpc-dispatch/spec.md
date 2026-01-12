# Capability: rpc-dispatch (Delta)

## MODIFIED Requirements

### Requirement: Global handler registry
The system SHALL maintain a process-wide registry keyed by `(protocol, serviceName)`.

The registry SHALL support at least two protocols:
- `grpc`
- `connectrpc`

The system SHALL expose protocol identifiers as stable exported constants via `rpcruntime.Protocol`:
- `rpcruntime.ProtocolGrpc` MUST be the identifier for `grpc`.
- `rpcruntime.ProtocolConnectRPC` MUST be the identifier for `connectrpc`.

#### Scenario: Protocol identifiers are stable and reusable
- **GIVEN** generated adaptor code and CGO-side Go code need to agree on protocol selection
- **WHEN** a caller sets `protocol` using `rpcruntime.ProtocolGrpc` or `rpcruntime.ProtocolConnectRPC`
- **THEN** the dispatch registry lookup SHALL route to the corresponding handler slot

---

### Requirement: Protocol selection is carried in context
The system SHALL provide helper functions in `rpcruntime` for carrying protocol selection in `context.Context`:
- `WithProtocol(ctx, protocol) context.Context`
- `ProtocolFromContext(ctx) (protocol, ok)`

Generated adaptor code SHALL use these helpers to determine which handler slot to lookup.

#### Scenario: Adaptor reads protocol from context
- **GIVEN** a `ctx` that was wrapped using `rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)`
- **WHEN** adaptor code performs a handler lookup
- **THEN** it SHALL route to the `grpc` handler slot for that `serviceName`
