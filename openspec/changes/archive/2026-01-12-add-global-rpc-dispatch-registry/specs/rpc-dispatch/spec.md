# Capability: rpc-dispatch

## Purpose
Provide a process-wide registry that allows generated adaptor code to register RPC service handlers and route calls to the correct handler implementation.

## ADDED Requirements

### Requirement: Global handler registry
The system SHALL maintain a process-wide registry keyed by `(protocol, serviceName)`.

The registry SHALL support at least two protocols:
- `grpc`
- `connectrpc`

#### Scenario: Register and lookup handler
- **GIVEN** a `serviceName = S`, a `protocol = P`, and a handler value `H`
- **WHEN** the handler is registered
- **THEN** the registry SHALL allow later lookup of `(P, S)` and return the same `H`

---

### Requirement: Independent grpc and connectrpc handlers
The system SHALL allow registering a `grpc` handler and a `connectrpc` handler for the same `serviceName` without conflict.

#### Scenario: Two protocols registered for one serviceName
- **GIVEN** a `serviceName = S`
- **WHEN** a `grpc` handler is registered for `S`
- **AND** a `connectrpc` handler is registered for `S`
- **THEN** both lookups SHALL succeed and return the respective handlers

---

### Requirement: Registration and replace semantics
The system SHALL provide a registration API that accepts `serviceName` and a handler value.
The system SHALL overwrite an existing handler for the same `(protocol, serviceName)` when registration is performed again.

#### Scenario: Replace registration succeeds
- **GIVEN** `(protocol=P, serviceName=S)` is already registered with handler `H1`
- **WHEN** registration is attempted again using handler `H2`
- **THEN** the registry SHALL contain `H2` for `(P, S)`

---

### Requirement: FullMethod conventions for routing
The system SHALL standardize method identifiers as `fullMethod` in the form `/Service/Method`.

#### Scenario: Generated code uses fullMethod constants consistently
- **GIVEN** a service `S` and method `M`
- **WHEN** adaptor code needs a stable identifier for routing/debugging
- **THEN** it SHOULD use the `fullMethod` form `/S/M`

---

### Requirement: Thread-safety
The registry and lookup APIs SHALL be safe for concurrent use.

#### Scenario: Concurrent invokes
- **GIVEN** a registered `(protocol, serviceName)` handler
- **WHEN** multiple goroutines perform lookups concurrently (including while a replace registration occurs)
- **THEN** the system SHALL not panic

## Notes
- Streaming methods are supported by storing the entire handler in the registry and letting generated adaptor code call the strongly-typed method after lookup.
