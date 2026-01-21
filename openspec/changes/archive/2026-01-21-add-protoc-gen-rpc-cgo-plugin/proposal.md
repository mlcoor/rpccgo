# Change: Add `protoc-gen-rpc-cgo` (CGO C ABI codegen)

## Why
当前仓库已有：
- `rpcruntime`：全局 handler registry、错误消息 registry（带 TTL）、协议选择。
- `protoc-gen-rpc-cgo-adaptor`：生成“Go 侧 adaptor API”，用于把（未来的）CGO `package main` 入口转发到已注册的 handler。

缺失组件：
- `protoc-gen-rpc-cgo`：生成 CGO `package main` 代码（`//export` C ABI），对外导出稳定的 C ABI 函数，并在内部调用 adaptor 生成的 Go API。

没有该插件时，动态库使用方无法通过 C ABI 调用 RPC handler（也无法通过统一错误检索 ABI 获取错误消息）。

## What Changes
- 新增 protoc 插件 `protoc-gen-rpc-cgo`（目录：`cmd/protoc-gen-rpc-cgo/`），支持生成 `package main` 的 CGO 导出代码。
- 新增能力 spec：`rpc-cgo-generator`（在本 change 中以 spec delta 形式定义）。
- 生成代码遵循：
  - 统一 ABI 命名（前缀 `Ygrpc_` + `Service` + `Method` 派生）。
  - 所有导出函数返回 `int`：`0` 成功，非 `0` 为错误 id。
  - 通过 `Ygrpc_GetErrorMsg(error_id, ...)` 进行错误消息检索（由 `rpc-runtime` 定义语义）。
  - 支持请求释放策略（`free_strategy`）与 Native 变体（`native`），其开关由 proto 文件内的 options 决定，并保证 Native/Binary 不混用。
- 在 `cgotest/` 增加端到端测试（C 程序调用生成的动态库），覆盖 Unary / Client-Streaming / Server-Streaming / Bidi-Streaming（Binary + Native）以及不同 adaptor 框架配置矩阵（`grpc`/`connect`/`connect_suffix`/`mix`）。
- 在 `cgotest/README.md` 记录如何运行测试，并补齐 build 脚本。

## Non-Goals
- 不在本 change 中改动 `rpcruntime` 的错误 TTL 或 handler registry 的语义。
- 不在本 change 中引入新的 RPC 框架类型（仅复用既有 adaptor 支持的框架）。
- 不在本 change 中定义跨进程/跨线程的 stream handle 语义（仅保证进程内、同一动态库实例内有效）。

## Impact
- Affected specs:
  - 新增：`rpc-cgo-generator`
  - 依赖：`rpc-cgo-adaptor`（Go API）、`rpc-runtime`（错误检索 ABI）、`rpc-dispatch`（协议选择）
- Affected code:
  - `cmd/protoc-gen-rpc-cgo/`（实现插件）
  - `cgotest/`（新增/扩展 C 端集成测试与 build 脚本）

## Open Questions
1) options 定义位置：仓库内存在 `proto/ygrpc/options/ygrpc/` 目录但暂无 `cgo` 相关 options 定义。是否在实现阶段补齐一个官方的 `cgo.proto`（例如定义 `ygrpc_cgo_native_default` / `ygrpc_cgo_native` / `ygrpc_cgo_req_free_default` / `ygrpc_cgo_req_free_method`），以避免业务 proto 重复定义？
2) 生成文件命名：本 change 采用与 adaptor 一致的命名规则：按 proto file 的 `GeneratedFilenamePrefix` 生成 `<fileName>_cgo.go`（一个 proto 输入文件对应一个输出文件，文件内包含该文件里的所有 services）。另生成独立的 `main.go`（包含 `func main()`）。
