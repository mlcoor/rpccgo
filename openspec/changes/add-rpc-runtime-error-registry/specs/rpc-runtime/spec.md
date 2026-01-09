# Capability: rpc-runtime (Delta)

## ADDED Requirements

### Requirement: Global error message registry with TTL
The runtime SHALL maintain a global mapping `errorId -> errorMsg(bytes)`.
- Each record SHOULD be retained for approximately 3 seconds and MAY be cleaned up after that.

#### Scenario: Error message expires
- **GIVEN** an `errorId` stored at time T
- **WHEN** time is later than T+3s and cleanup has occurred
- **THEN** `Ygrpc_GetErrorMsg(errorId, ...)` SHALL return `1` indicating not found/expired

---

### Requirement: `Ygrpc_GetErrorMsg` ABI
The system SHALL provide the following C ABI for retrieving error messages.
The ABI implementation SHALL delegate error lookup to `rpc-runtime` (this capability), while the exported symbol itself is expected to live in a CGO `package main` (e.g., generated CGO code).

```c
typedef void (*FreeFunc)(void*);

int Ygrpc_GetErrorMsg(int error_id, void** msg_ptr, int* msg_len, FreeFunc* msg_free);
```

Semantics:
- return `0` when found and outputs are set
- return `1` when not found/expired

Buffer rules:
- The returned `msg_ptr` SHALL be allocated using `malloc`-compatible allocation.
- The returned `msg_free` SHALL be a callable free function compatible with freeing `msg_ptr`.

#### Scenario: Found message returned as malloc buffer
- **GIVEN** an existing error message for `error_id`
- **WHEN** `Ygrpc_GetErrorMsg` is called
- **THEN** it SHALL allocate output buffer using `malloc`-compatible allocation
- **AND** set `msg_free` to a callable free wrapper compatible with standard `free`
