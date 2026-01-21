# Change: Add runtime error registry and `Ygrpc_GetErrorMsg` ABI

## Why
当前项目缺少一个稳定、跨语言可用的错误模型。
在 CGO 导出函数返回 `errorId != 0` 的情况下，C 侧需要一个统一 ABI 来取回错误信息，并且错误信息需要有明确的生命周期与过期策略。

## What Changes
- 新增 1 个能力的 spec delta：
  - `rpc-runtime`: 定义全局错误 registry（`errorId -> errorMsg(bytes)`，带 TTL）以及 `Ygrpc_GetErrorMsg` C ABI。

## Non-Goals
- 不在本 change 中实现 `protoc-gen-*` 两个插件。
- 不在本 change 中定义/实现 handler 注册、procedure 路由、Binary/Native 调用 ABI。
- 不在本 change 中处理 streaming。
- 不在本 change 中单独提供一个手写的 `package main` CGO 导出二进制；`Ygrpc_GetErrorMsg` 的导出由后续生成的 CGO 代码引入并调用本 runtime 库。

## Impact
- Affected specs (new deltas): `rpc-runtime`.
- Affected code (apply stage): 新增 `rpcruntime/` 包（作为库被 CGO 生成代码调用）。
- Compatibility:
  - 新项目：无兼容性负担。
  - 后续扩展：此错误 ABI 作为基础设施，后续其它 ABI/生成器可复用。
