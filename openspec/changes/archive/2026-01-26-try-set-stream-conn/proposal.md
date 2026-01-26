## Why

当前 `rpcruntime/connect_stream.go` 的 `Set*StreamConn` 在调用方传入 `nil` 或遇到反射布局/类型问题时会 `panic`。
这对部分调用场景不够友好，也不利于上层做可控的错误处理与降级。

## What Changes

- 新增 `TrySetClientStreamConn` / `TrySetServerStreamConn` / `TrySetBidiStreamConn`：遇到无效输入或内部异常时返回 `error`，不触发 `panic`
- 保持现有 `Set*StreamConn` 行为不变（继续 `panic`，作为“编程错误”的 hard fail）
- 增加单元测试覆盖 `TrySet*`：
	- `nil` stream 返回错误
	- 正常情况下可正确写入 `conn`

## Capabilities

### New Capabilities

- `try-set-stream-conn`: 为 Connect stream 的 `conn` 注入提供非 panic 的安全 API（返回 `error`）

### Modified Capabilities

（无）

## Impact

- `rpcruntime/connect_stream.go`: 增加 `TrySet*StreamConn` 实现（复用现有反射写入逻辑）
- `rpcruntime/connect_stream_test.go`: 增加针对 `TrySet*` 的测试用例
