# Design: RPC CGO adaptor code generator

## Context
- 运行时全局注册表已落地在 `rpcruntime`（见 `RegisterGrpcHandler`/`RegisterConnectHandler` 与 `LookupGrpcHandler`/`LookupConnectHandler`）。
- adaptor 生成器的职责是把“从注册表取 handler + 强类型调用具体方法”的样板代码生成出来，供 CGO 侧的 Go 代码调用。

## Goals
- 生成的调用入口是 **Go API（普通函数）**，供 CGO 生成的 `package main` 代码调用。
- 方法参数使用 proto 的 Go 消息结构体，保持与 `.proto` 定义一致、类型安全。
- 每次调用都显式携带 `context.Context`（透传 cancellation/deadline/trace 等），并通过 `ctx` 传递“框架选择”（`grpc` / `connectrpc`）。
- adaptor 负责：
  - 从 `ctx` 中解析 protocol 并选择 lookup 函数
  - 断言 handler 类型
  - 调用具体方法

同时，支持通过 protoc plugin option 控制生成哪些框架的 adaptor 代码（默认全生成）。

## Non-Goals
- 不生成 `//export` C ABI 函数，不处理 `errorId`（由 `protoc-gen-rpc-cgo` 负责）。
- 不引入反射扫描 handler 方法；仍坚持 handler-centric 模式。

## Key Decisions
### 1) Protocol carried in context
生成函数签名不再携带独立的 `protocol` 参数；调用方通过 `ctx` 传递 protocol。

为避免 context key 冲突并让生成代码稳定复用，运行时提供 helper：
- `rpcruntime.WithProtocol(ctx, protocol) context.Context`
- `rpcruntime.ProtocolFromContext(ctx) (rpcruntime.Protocol, bool)`

理由：
- 协议选择是一次调用的“调用元信息”，用 `context.Context` 传递更符合 Go 的惯例。
- 生成代码不暴露额外参数，接口更稳定，便于未来扩展更多调用元信息。

### 2) Codegen frameworks controlled via plugin options
生成器支持通过 protoc plugin option 控制生成哪些框架的 adaptor 代码：
- 默认：同时生成 grpc + connectrpc。
- 可选：只生成 grpc 或只生成 connectrpc（用于缩小依赖/减少生成物）。

### 2) Streaming API as staged calls + callbacks
为满足 CGO 场景下“参数尽量是 proto 消息结构体、接口可控、跨框架一致”的目标，streaming 方法不直接暴露框架 stream 类型。

生成器将 streaming 方法转为一组更适合调用侧的 Go 函数：

- client-streaming：拆分为三个阶段
  - `Start(ctx, ...) -> streamHandle(uint64)`
  - `Send(streamHandle, req *Req) error`（可多次调用）
  - `Finish(streamHandle) (*Resp, error)`（结束并拿到最终响应）

- server-streaming：改为回调消费
  - `Call(ctx, req *Req, onRead func(*Resp) bool, onDone func(error)) error`
    - `onRead` 每收到一条响应调用一次；返回 false 表示停止继续读取。
    - `onDone` 在流结束或出错时调用一次。

其中：
- `streamHandle` 为进程内不透明句柄（`uint64`），由 adaptor 内部维护映射与生命周期；`Finish` 后立即失效。
- `onRead` 返回 false 时，adaptor 需要像 gRPC 常见用法一样通过取消该次调用的 context（或等价终止手段）来尽快停止底层流，避免 goroutine/连接资源悬挂。

该设计的核心取舍：
- adaptor 内部负责把“分段调用/回调”桥接到真实 handler（grpc/connectrpc）的 streaming 实现。
- adaptor 内部需要管理 goroutine 与资源清理（避免泄漏），并将错误确定性返回给调用方。

## Error Semantics
adaptor 生成代码返回普通 `error`，并区分以下可判定错误（用于上层转换为 errorId）：
- 未知 protocol
- service 未注册（lookup ok=false）
- handler 类型不匹配（类型断言失败）

## Notes
- bidi-streaming 将在 apply 阶段按同样思路扩展为 `Start/Send/CloseSend` + `onRead/onDone`（具体函数集合以实现阶段任务细化为准）。
