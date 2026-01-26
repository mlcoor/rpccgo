## Context

`rpcruntime/connect_stream.go` 通过 `reflect` 将 `connect.ClientStream/ServerStream/BidiStream` 的未导出字段 `conn` 注入为 `connect.StreamingHandlerConn`。
当前公开 API `Set*StreamConn` 在如下情况下会 `panic`：

- 传入 `nil` stream
- Connect stream 结构布局校验失败（`mustCheckConnectStreamLayout`）
- 反射设置字段过程中出现不可预期情况

在一些调用路径中，调用方更希望以 `error` 处理失败，而不是让进程崩溃。

## Goals / Non-Goals

**Goals:**

- 提供非 `panic` 的 `TrySet*StreamConn` API：失败时返回 `error`
- 对 `nil` stream 等常见无效输入给出明确的 `error`
- 为新增行为提供单元测试覆盖

**Non-Goals:**

- 不改变现有 `Set*StreamConn` 的语义与签名
- 不重写或替换反射注入方案（保持实现改动最小）
- 不引入新的外部依赖

## Decisions

### Decision 1: 通过 recover 将 panic 转为 error

`TrySet*StreamConn` 内部调用现有 `Set*StreamConn`，并使用 `defer` + `recover` 捕获所有 `panic`，将其转换为 `error` 返回。
这样可以最大化复用既有逻辑，并保证即使未来内部实现新增 `panic` 路径，`TrySet*` 也能保持“不崩溃”的契约。

### Decision 2: 显式处理 nil stream

在进入 `Set*StreamConn` 之前先对 `nil` 进行判断并返回 `error`，避免出现带有 `panic:` 前缀的错误信息，使 API 语义更直接。
