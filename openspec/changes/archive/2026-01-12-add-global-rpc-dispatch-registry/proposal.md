# Change: Add global RPC dispatch registry

## Why
当前生成链路中，CGO 侧代码需要通过 adaptor 访问具体的 Go 端 RPC 实现，但缺少一个统一、可复用的“全局注册 + 查找 + 调用”的机制。

现状（见仓库 README 的 mermaid 图示）假定存在 `GlobalRegister`：adaptor 能通过它拿到具体实现并完成调用，但仓库尚未定义该机制的运行时 API 与行为规范。

## What Changes
- 新增运行时能力 `rpc-dispatch`：提供全局的“按 serviceName 注册 handler、按 serviceName 查找并路由调用”的机制。
- 支持两套独立实现并行注册：同一个 `serviceName` 下，分别注册 `grpc` handler 与 `connectrpc(simple)` handler，adaptor 可按调用来源路由到对应实现。
- 支持替换（replace）语义：允许在运行时用新 handler 覆盖旧 handler（用于热更新/注入/测试）。
- 提供 adaptor 友好的最小注册 API：`RegisterHandler(serviceName string, handler any, ...)`（并带可选参数指定 `grpc`/`connectrpc` 维度）。
- 明确并规范化错误语义：未注册/handler 类型不匹配/替换策略等情况的返回错误。

## Non-Goals
- 不引入网络传输层（不做 client/server、连接、编解码）。
- 不试图复刻 grpc interceptor、reflection、health 等完整生态。

## Impact
- Affected specs:
  - `rpc-dispatch`（新增 capability）
- Affected code (apply 阶段实现时):
  - `rpcruntime/`：新增全局 registry 与 dispatch API
  - 注：adaptor 生成与路由调用将通过后续单独 change 提案完成（本 change 不包含实现）。

## Compatibility
- 该 change 以新增能力为主，不破坏现有 `rpc-runtime` error registry 行为。
- 对外 API 以最小集合起步，后续通过新增 API 扩展（避免破坏性修改）。
