## 1. rpcruntime API

- [x] 1.1 在 `rpcruntime/connect_stream.go` 增加 `TrySetClientStreamConn`，对 `nil` stream 返回 `error`，并将内部 `panic` 转为 `error`
- [x] 1.2 增加 `TrySetServerStreamConn`，语义同上
- [x] 1.3 增加 `TrySetBidiStreamConn`，语义同上

## 2. Tests

- [x] 2.1 在 `rpcruntime/connect_stream_test.go` 增加 `TrySet*` 的测试：nil stream 返回 error
- [x] 2.2 增加 `TrySet*` 的测试：正常情况下可正确设置 conn

## 3. Verify

- [x] 3.1 运行 `go test ./...`
