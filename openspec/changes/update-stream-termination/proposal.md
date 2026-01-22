# Change: Update stream termination to avoid send/close panic

## Why
当前 runtime 在并发场景下对 sendCh 的 close 与发送可能触发 panic，影响稳定性与可预测性。

## What Changes
- 采用与 grpc 一致的“半关闭”语义：CloseSend 仅标记发送结束，不取消 ctx。
- 发送侧结束后接收端以 `io.EOF` 表示流完成，不依赖 close(sendCh)。
- 明确 CloseSend/Finish 与 Send 的并发规则（不可并发），并在实现中串行化或返回确定性错误以避免 panic。

## Impact
- Affected specs: rpc-cgo-adaptor
- Affected code: rpcruntime/stream_handle.go 与所有流相关 adaptor 代码
