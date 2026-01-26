# try-set-stream-conn Specification

## Purpose
TBD - created by archiving change try-set-stream-conn. Update Purpose after archive.
## Requirements
### Requirement: Non-panic stream conn injection

系统 MUST 提供一组不会 `panic` 的 API，用于为 Connect 的 stream 类型注入 `connect.StreamingHandlerConn`。
当输入无效或内部发生异常时，API MUST 以 `error` 形式返回失败原因，让调用方可以选择处理或降级。

#### Scenario: Nil stream returns error

- **WHEN** 调用 `TrySetClientStreamConn(nil, conn)`
- **THEN** MUST 返回非空 `error`
- **AND** 不发生 `panic`

- **WHEN** 调用 `TrySetServerStreamConn(nil, conn)`
- **THEN** MUST 返回非空 `error`
- **AND** 不发生 `panic`

- **WHEN** 调用 `TrySetBidiStreamConn(nil, conn)`
- **THEN** MUST 返回非空 `error`
- **AND** 不发生 `panic`

#### Scenario: Valid stream sets conn successfully

- **WHEN** 调用 `TrySetClientStreamConn(stream, conn)` 且 `stream` 为非 nil 的 `*connect.ClientStream[T]`
- **THEN** MUST 返回 `nil` error
- **AND** `stream` 内部的 `conn` 字段等于传入的 `conn`

- **WHEN** 调用 `TrySetServerStreamConn(stream, conn)` 且 `stream` 为非 nil 的 `*connect.ServerStream[T]`
- **THEN** MUST 返回 `nil` error
- **AND** `stream` 内部的 `conn` 字段等于传入的 `conn`

- **WHEN** 调用 `TrySetBidiStreamConn(stream, conn)` 且 `stream` 为非 nil 的 `*connect.BidiStream[Req, Res]`
- **THEN** MUST 返回 `nil` error
- **AND** `stream` 内部的 `conn` 字段等于传入的 `conn`

