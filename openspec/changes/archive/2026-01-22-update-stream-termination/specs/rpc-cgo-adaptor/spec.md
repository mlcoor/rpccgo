## MODIFIED Requirements

### Requirement: Stream handle lifecycle is well-defined
The generated adaptor code SHALL ensure stream handles are safe and deterministic:
- `Start` SHALL return a non-zero `uint64` handle when successful.
- `Send` and `Finish` SHALL return a non-nil error when the provided handle is unknown, already finished, or otherwise invalid.
- After `Finish` returns (success or error), the handle SHALL become invalid and MUST NOT be reused.
- CloseSend/Finish SHALL perform a half-close (send side only) and MUST NOT cancel the stream context.
- Concurrent `Send` and `CloseSend/Finish` calls SHALL NOT panic; the implementation SHALL serialize or return a deterministic error.
- After send-side half-close, the receive path SHALL return `io.EOF` without relying on send channel close.

#### Scenario: Concurrent Send and CloseSend do not panic
- **GIVEN** a valid stream handle from `Start`
- **WHEN** `Send` and `CloseSend/Finish` are invoked concurrently
- **THEN** no panic occurs
- **AND** the implementation either serializes or returns a deterministic error
- **AND** the receive path terminates with `io.EOF`
