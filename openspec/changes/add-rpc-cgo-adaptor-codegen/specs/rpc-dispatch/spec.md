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
- **GIVEN** generated adaptor code and `rpcruntime` need stable protocol identifiers
- **WHEN** adaptor/runtime code uses `rpcruntime.ProtocolGrpc` or `rpcruntime.ProtocolConnectRPC`
- **THEN** handler lookup via `LookupGrpcHandler` / `LookupConnectHandler` SHALL route to the corresponding handler slot
