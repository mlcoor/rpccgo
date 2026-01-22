## 1. Implementation
- [x] 调整 rpcruntime 的流结束机制，移除 sendCh close，采用半关闭标记（不取消 ctx）
- [x] 更新 adaptor 的流接收逻辑：发送结束后返回 io.EOF
- [x] 明确 CloseSend/Finish 与 Send 的并发规则，并在实现中串行化或返回错误
- [x] 补充/更新并发 Send/CloseSend 的单测与回归测试

## 2. Documentation
- [x] 更新实现相关注释与说明，明确流结束语义
