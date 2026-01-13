## 背景（Context）
目前生成的 CGO adaptor 会根据插件参数在生成期硬编码 dispatch protocol，导致即使 `rpcruntime` 能同时承载 `grpc`/`connectrpc` 的 handler，调用方依然无法在运行时做选择。

本变更引入一个统一且稳定的机制：把 protocol 选择携带在 `context.Context` 中，并调整 codegen，让通用 adaptor 能在运行时动态选择合适的 protocol。

## 目标 / 非目标（Goals / Non-Goals）
- Goals:
  - 允许调用方通过 `context.Context` 显式指定 `grpc` 或 `connectrpc`。
  - 当未指定时，按配置的有序列表进行 fallback。
  - 在未显式指定时保持默认行为不变。
  - 当路由失败时返回确定性、可调试的错误。
- Non-Goals:
  - 引入 `grpc` 与 `connectrpc` 之外的新 protocol。
  - 改变底层 handler registry 的模型或注册 API。

## 建议 API 与术语（Proposed API & Terminology）
- 统一使用术语 `protocol`，并与 `rpcruntime.Protocol` 1:1 对齐。
- 选中的 protocol 通过 `ctx` 携带，使用稳定 key：
  - `rpcruntime.ContextKeyProtocol`
- 建议提供 helper（实现阶段）：
  - `rpcruntime.WithProtocol(ctx, rpcruntime.Protocol) context.Context`
  - `rpcruntime.ProtocolFromContext(ctx) (rpcruntime.Protocol, bool)`

## 选择算法（Selection Algorithm：Universal Adaptor）
Inputs:
- `ctx`: 可能包含显式 protocol。
- `supported`: 由插件参数 `protocol` 派生的有序列表。

Codegen cases:
- `supported` 仅包含一个 protocol：生成的代码 SHOULD 直接 lookup 该 protocol 的 handler（不生成 switch/遍历其他协议的逻辑）。
- `supported` 包含多个 protocol：生成的代码 SHOULD 同时支持“ctx 显式指定”和“ctx 未指定时按列表 fallback”。

Algorithm:
1. 如果 `ctx` 中包含 protocol：
   - 若该值不在 `supported` 中，返回 `ErrUnknownProtocol`。
   - 否则仅按该 protocol 做一次 dispatch。
2. 如果 `ctx` 中不包含 protocol：
   - 按 `supported` 顺序遍历。
   - 对每个 protocol 尝试查找 `(protocol, serviceName)` 的 handler。
   - 首次命中则类型断言并调用。
3. 若所有查找都失败，返回 `ErrServiceNotRegistered`。

## 插件参数解析（Generator Option Parsing）
本变更实现建议：使用 `protocol` 选项作为唯一入口（逗号分隔、有序列表）。

解析规则（建议）：
- 输入来源：protoc plugin 参数 `protocol`。
- 以逗号分隔；对每个 token 执行 `TrimSpace + ToLower`。
- 只允许 `grpc` / `connectrpc`。
- 去重但保持首次出现顺序。
- 省略或解析后为空时，默认列表为 `[connectrpc]`。

当 `protocol` 包含 `connectrpc` 时，插件额外支持一个可选参数用于适配 connect-go 的独立 package 输出：

- `connect_package_suffix`：connect handler interface 所在 Go package 的“子包后缀”。
  - 默认值为空字符串。
  - 为空时：默认 handler interface 位于当前 Go package（等价于 connect-go `package_suffix=""` 的同包生成模式）。
  - 非空时：生成的代码在做 connect 分支类型断言时，使用 connect-go 的生成规则推导子包：其 import path 为 `<current-import-path>/<current-go-package-name><connect_package_suffix>`，并在该子包中引用 `<Service>Handler` 接口类型。

实现细节（便于落地）：
- `current-import-path` 可直接使用 `file.GoImportPath`。
- `current-go-package-name` 可直接使用 `file.GoPackageName`（adaptor 与 `*.pb.go` 同包生成时）。
- 当 `connect_package_suffix != ""` 时：
  - `connectSubpackageName := string(file.GoPackageName) + connect_package_suffix`
  - `connectHandlerImportPath := path.Join(string(file.GoImportPath), connectSubpackageName)`
  - 生成 connect handler interface 的 GoIdent 时，使用 `protogen.GoImportPath(connectHandlerImportPath)` 作为 import path。
  - 同时建议与 connect-go 保持一致：对 `connect_package_suffix` 做 Go identifier 校验（例如 Go `token.IsIdentifier`）。

## 生成代码结构（Generated Code Shape）
为了避免在每个 method 内复制协议选择逻辑，生成代码 SHOULD 在每个 service 维度生成一个内部 helper：

```go
// 返回：最终选中的 protocol + 对应 handler。
// err != nil 表示：ctx 指定了不支持协议，或已无可用 handler。
func <Service>_lookupHandler(ctx context.Context) (rpcruntime.Protocol, any, error)
```

### helper 的生成策略
- 当 `supported` 为单协议：生成直线逻辑，只调用一次对应的 `Lookup*Handler`。
- 当 `supported` 为多协议：生成“ctx 显式指定” + “ctx 缺省 fallback”两段逻辑。

多协议 helper 伪代码：

```go
if p, ok := rpcruntime.ProtocolFromContext(ctx); ok {
  // 显式 protocol：只按该协议查找
  switch p {
  case rpcruntime.ProtocolGrpc:
    // 仅当 supported 包含 grpc 才生成该 case，否则走 default
    if h, ok := rpcruntime.LookupGrpcHandler(serviceName); ok { return p, h, nil }
    return p, nil, rpcruntime.ErrServiceNotRegistered
  case rpcruntime.ProtocolConnectRPC:
    if h, ok := rpcruntime.LookupConnectHandler(serviceName); ok { return p, h, nil }
    return p, nil, rpcruntime.ErrServiceNotRegistered
  default:
    return p, nil, rpcruntime.ErrUnknownProtocol
  }
}

// 未显式 protocol：按 supported 顺序 fallback
for _, p := range supported {
  if p == rpcruntime.ProtocolGrpc {
    if h, ok := rpcruntime.LookupGrpcHandler(serviceName); ok { return p, h, nil }
  } else if p == rpcruntime.ProtocolConnectRPC {
    if h, ok := rpcruntime.LookupConnectHandler(serviceName); ok { return p, h, nil }
  }
}

return "", nil, rpcruntime.ErrServiceNotRegistered
```

注意：在 codegen 层面，helper 的 switch/case 和 fallback 循环都 MUST 只包含 `supported` 中出现过的协议；这样当 `supported` 单值时，生成物不会引用其他协议相关的 lookup / interface，从而满足“单协议只生成相关代码”的需求。

## 方法级调用分支（Unary / Streaming）

### Unary
Unary 方法的主体 SHOULD：
1) 调用 `<Service>_lookupHandler(ctx)` 获取 `(protocol, handler)`。
2) 按 protocol 做类型断言与实际调用：
   - `grpc` → 断言为 `<Service>Server` 并调用 `svc.Method(ctx, req)`
   - `connectrpc` → 断言为 `<Service>Handler` 并调用 `svc.Method(ctx, req)`

### Streaming
Streaming 方法/Start 方法中同样先选 `(protocol, handler)`，然后按 protocol 分支：
- gRPC 分支：使用现有 adaptorStream + goroutine 的调用方式。
- connectrpc 分支：使用现有 rpcruntime 的 stream helper（`NewClientStream` / `NewServerStream` / `NewBidiStream` 等）。

额外说明：
- 只有当 `supported` 包含 `grpc` 时，才生成 grpc 的 stream adaptor types（`*_Server` 的 adaptor struct）。
- `AllocateStreamHandle(ctx, protocol)` 的 protocol 参数 MUST 使用“最终选中的 protocol”。

## 测试方案（cgotest 生成与混合注册）
### 目录与生成物
`cgotest/` 下建议维持三套目录：
- `cgotest/connect/`：单协议 `protocol=connectrpc`
- `cgotest/grpc/`：单协议 `protocol=grpc`
- `cgotest/mix/`：多协议 `protocol=grpc|connectrpc`（用于混合注册语义测试）

### 解决 connectrpc/grpc 生成代码冲突
问题：如果在同一个 Go package 同时运行 `protoc-gen-go-grpc` 与 `protoc-gen-connect-go`，两者都会生成同名的 client 符号（典型是 `New<Service>Client`），从而导致 Go 编译期符号重复。

更贴近“通过包隔离实现无缝切换”的建议策略是：
- `cgotest/all/` 的 base package 同时生成：
  - message 类型（`protoc-gen-go`）
  - gRPC stubs（`protoc-gen-go-grpc`，同包，包含 `<Service>Server`）
  - adaptor（`protoc-gen-rpc-cgo-adaptor`，同包）

- 让 `protoc-gen-connect-go` 按其默认模式生成到“独立的 connect 子 package”（默认 package_suffix），避免与 go-grpc 在同包内产生命名冲突。
- 生成 adaptor 时通过 `connect_package_suffix` 指定 connect handler interface 所在子包后缀，从而无需额外手写 type alias 文件。

这样可以达到：
- grpc client 与 connect client 分属不同 package，不再冲突。
- connect 侧 handler interface 仍然使用 base messages（connect-go 的默认设计是“connect 专用包 import base types”），adaptor 调用时参数类型一致。
- 混合注册测试可以真正做到“同一份 adaptor 代码，根据 ctx 选择 grpc/connect 并无缝切换”。

### 生成脚本建议（实现阶段落地到 build-all.sh）
`cgotest/build-all.sh` SHOULD 一键生成三套目录（connect/grpc/all），其中 `all` 的生成仅需要：
- `protoc-gen-go`（messages）
- `protoc-gen-go-grpc`（grpc stubs，同包）
- `protoc-gen-connect-go`（connect stubs，独立子包）
- `protoc-gen-rpc-cgo-adaptor`（adaptor，同包）

`all` 的核心命令形态（示意）：

```bash
protoc -Iproto \
  --go_out=./all --go_opt=paths=source_relative,... \
  --go-grpc_out=./all --go-grpc_opt=paths=source_relative,... \
  --connect-go_out=./all --connect-go_opt=paths=source_relative,simple=true,... \
  --rpc-cgo-adaptor_out=./all \
  --rpc-cgo-adaptor_opt=paths=source_relative,protocol=grpc|connectrpc,... \
  ./proto/unary.proto
```

说明：
- connect-go 在 `all` 中必须生成到独立子包（不要设置 `package_suffix=""` 使其回退到同包生成），以避免与 go-grpc 的 client 符号冲突。
- 这里建议只对 `unary.proto` 做混合语义测试，以最小化与 streaming 相关的额外接口/类型依赖。

## 额外测试：connect_suffix
为覆盖“connect-go 使用非空 package_suffix（独立包）时 adaptor 仍可正常工作”的场景，建议新增 `cgotest/connect_suffix/`：
- connect-go 使用默认 package_suffix 生成到独立 connect 子包。
- adaptor 使用 `protocol=connectrpc` 且传入 `connect_package_suffix=<suffix>`（例如 `connect`）。

该测试用于验证：即使 connect handler interface 不在当前 package 中，adaptor 仍可正确做类型断言并调用。

### 混合注册测试用例（实现阶段落地到 cgotest/all）
在 `cgotest/all` 的测试中：同一个 `serviceName` 同时存在 grpc/connectrpc 两种注册可能，重点覆盖协议选择语义：

1) ctx 显式指定 grpc 但仅注册 connectrpc → 期望错误（不 fallback）
- 使用 `rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)`
- 仅调用 `rpcruntime.RegisterConnectHandler(serviceName, handler)`
- 调用生成的 adaptor entrypoint → 返回 `ErrServiceNotRegistered`（或等价错误）

2) ctx 不指定 + 注册 connectrpc → 期望 fallback 命中 connectrpc
- ctx 不设置 protocol
- 注册 connect handler
- adaptor 在 `protocol=grpc|connectrpc` 下先尝试 grpc lookup 失败，再尝试 connect lookup 成功

3) 单协议 `protocol=grpc` + 注册 connectrpc → 期望错误（不 lookup connectrpc）
- 该用例可以放在 `cgotest/grpc`（grpc-only 生成物）中：仅注册 connect handler，然后调用 adaptor → 必须返回错误

注：第 3 条的“未 lookup connectrpc”无法直接从黑盒判定调用次数，但通过“如果 lookup 了就会成功、实际却返回错误”可作为足够的行为验证。

## 插件参数与生成文件（Plugin Option & Generated Files）
- 插件参数：`protocol` 为逗号分隔列表。
  - 示例：`protocol=grpc|connectrpc`。
  - 省略时默认列表为 `connectrpc`。

每个 proto 输入文件的生成输出：
- 仅生成：`*_cgo_adaptor.go`（通用入口；CGO 侧最终调用此处的函数）。

文件不因协议拆分；生成代码仅在函数体内部根据 `protocol` 选项生成不同的查找/选择逻辑。

## 错误策略（Error Strategy）
- `ctx` 指定未知/不支持的 protocol：`rpcruntime.ErrUnknownProtocol`。
- 显式 protocol 但未注册 handler：`rpcruntime.ErrServiceNotRegistered`。
- fallback 遍历列表仍未命中任何 handler：`rpcruntime.ErrServiceNotRegistered`。
- handler 类型断言失败：`rpcruntime.ErrHandlerTypeMismatch`。

## 权衡（Trade-offs）
- Pros:
  - 在保持稳定 protocol 标识符的前提下支持运行时选择。
  - 默认行为不变。
- Cons:
  - 配置多个 protocol 时会生成更多文件（代码体积增加）。
  - 需要在 `rpcruntime` 中提供稳定的 context key 以及少量 helper API。
