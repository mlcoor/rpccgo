# Change: Add RPC CGO adaptor code generator

## Why
当前仓库已经具备了运行时全局注册表（`rpcruntime` 内的 `Register*Handler`/`Lookup*Handler`），但用于“把 CGO 侧的 Go 调用转发到具体 RPC 实现”的 adaptor 代码生成器仍为空（`cmd/protoc-gen-rpc-cgo-adaptor/main.go` 仅有 `package main`）。

这导致：即使业务侧已经把 grpc/connectrpc 的 service handler 注册进全局注册表，CGO 侧仍缺少一层**类型安全、与 proto 消息结构体对齐**的调用入口。

## What Changes
- 新增能力规范（spec delta）：`rpc-cgo-adaptor`
  - 定义 `protoc-gen-rpc-cgo-adaptor` 的生成产物形态与调用语义。
  - 生成的方法签名使用 proto 生成的 Go 消息结构体（例如 `*pb.Req` / `*pb.Resp`），并为 streaming 方法提供 staged/callback 形式的类型安全 API（不要求调用方直接持有框架 stream 对象）。
  - 每个生成的方法都接受 `context.Context` 作为第一个参数（用于 cancellation/deadline/trace 等透传）。框架选择（`grpc` / `connectrpc`）由生成器参数在生成阶段确定，不通过 `ctx` 传递。
  - 生成器支持通过 protoc plugin 参数选择生成哪种框架的 adaptor 代码（默认生成 connectrpc）。推荐形态：`--rpc-cgo-adaptor_opt=framework=grpc|connectrpc`。
  - Connect 框架仅支持 Simple API 模式（使用 `connect-simple=true` 生成的 handler 接口）。
- 补充 `rpc-dispatch` 规范（spec delta）：明确 protocol 选择使用 `rpcruntime.Protocol` 及其稳定常量值，以便生成器和调用方共享同一组枚举。

## Non-Goals
- 不在本 change 中实现 `protoc-gen-rpc-cgo`（生成 `//export` C ABI 入口与 errorId 处理的生成器）。
- 不在本 change 中引入网络层/客户端逻辑；只负责从全局注册表查找 handler 并调用。
- 不在本 change 中定义跨语言 C ABI 的函数签名（该部分由 `protoc-gen-rpc-cgo` 的后续 change 负责）。

## Impact
- Affected specs:
  - `rpc-cgo-adaptor`（新增 capability）
  - `rpc-dispatch`（补充 protocol 枚举与稳定性约定）
- Affected code (apply stage):
  - `cmd/protoc-gen-rpc-cgo-adaptor/`：实现生成器
  - 可能新增少量共享的生成辅助包（如 `internal/codegen/...`），用于可测试性

## Compatibility
- 以新增能力为主，不改变现有 `rpcruntime` dispatch registry 的对外行为。
- 生成代码依赖 `rpcruntime.ProtocolGrpc` / `rpcruntime.ProtocolConnectRPC` 的稳定值（当前实现为字符串常量）。
