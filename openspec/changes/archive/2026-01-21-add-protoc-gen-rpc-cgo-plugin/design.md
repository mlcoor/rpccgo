# Design: `protoc-gen-rpc-cgo` (CGO C ABI generator)

## Overview
`protoc-gen-rpc-cgo` 的职责是生成一个可用于 `go build -buildmode=c-shared` 的 `package main` 代码集合：
- 对外：导出稳定的 C ABI（`//export`）函数供动态库调用方使用。
- 对内：调用 `protoc-gen-rpc-cgo-adaptor` 生成的 Go API，把请求路由到 `rpcruntime` 已注册的 handler。
- 统一错误模型：所有导出函数返回 `int`（`0`=成功，非 0=错误 id），错误消息通过 `Ygrpc_GetErrorMsg` 查询。

## Inputs
- Protobuf service/method 定义（含 streaming 类型）。
- Proto 文件内 options（FileOptions / MethodOptions 扩展）决定是否生成 Native 变体与 TakeReq 变体。
  - `ygrpc_cgo_req_free_default`（FileOptions, int32）：`0=none, 1=take_req, 2=both`（默认 `0`）
  - `ygrpc_cgo_req_free_method`（MethodOptions, int32）：同上；若存在则覆盖 file-level default
  - `ygrpc_cgo_native_default`（FileOptions, int32）：`0=disable, 1=enable`（默认 `0`）
  - `ygrpc_cgo_native`（MethodOptions, int32）：同上；若存在则覆盖 file-level default

覆盖顺序：method option（若存在）优先于 file option；都不存在则使用默认值。

## Outputs
输出目录由 `rpc-cgo_out` 选项指定。生成结构：

```
<output_dir>/
├── main.go
├── <fileName>_cgo.go
└── ...
```

约束：
- 所有文件 `package main`。
- `main.go` 含 `func main()`（可以为空实现）。

## ABI Naming
导出符号命名与 adaptor 命名风格对齐：以 `Service` 与 `Method` 拼接派生，并统一加前缀 `Ygrpc_`。

Unary:
- Binary: `Ygrpc_{Service}_{Method}` / `Ygrpc_{Service}_{Method}_TakeReq`
- Native: `Ygrpc_{Service}_{Method}_Native` / `Ygrpc_{Service}_{Method}_Native_TakeReq`

Client-Streaming:
- Binary: `Ygrpc_{Service}_{Method}Start` / `Send` / `Send_TakeReq` / `Finish`
- Native: `Ygrpc_{Service}_{Method}Start_Native` / `Send_Native` / `Send_Native_TakeReq` / `Finish_Native`

Server-Streaming:
- Binary: `Ygrpc_{Service}_{Method}`（callback 模式）/ `Ygrpc_{Service}_{Method}_TakeReq`
- Native: `Ygrpc_{Service}_{Method}_Native` / `Ygrpc_{Service}_{Method}_Native_TakeReq`

Bidi-Streaming:
- Binary: `Ygrpc_{Service}_{Method}Start` / `Send` / `Send_TakeReq` / `CloseSend`
- Native: `Ygrpc_{Service}_{Method}Start_Native` / `Send_Native` / `Send_Native_TakeReq` / `CloseSend_Native`

所有导出函数在 C preamble 的注释中标注用途、参数含义与内存规则。

## Error Model
- 每个导出 ABI 函数返回 `int`：
  - `0`: success
  - `!=0`: error id
- 当 adaptor 调用返回 `error` 时，生成代码 SHALL 调用 `rpcruntime.StoreError(err)` 将错误消息写入 registry，并返回其 id（转换为 `int`）。
- 错误消息检索使用 `rpc-runtime` 中定义的 ABI：

```c
typedef void (*FreeFunc)(void*);
int Ygrpc_GetErrorMsg(int error_id, void** msg_ptr, int* msg_len, FreeFunc* msg_free);
```

为避免多重定义，`FreeFunc` 和 `Ygrpc_GetErrorMsg` 的声明使用 `#ifndef` 宏保护。

## Request Free Strategy (`free_strategy`)
- `none`: 不生成 `*_TakeReq` 变体，且导出函数不接管请求缓冲区。
- `take_req`: 仅生成 `*_TakeReq` 变体，导出函数接收 `(reqPtr, reqLen, reqFree)`，并在完成后调用 `reqFree(reqPtr)`。
- `both`: 同时生成两种变体。

注意：Native 与 TakeReq 可组合，生成 `_Native_TakeReq`。

## Native Eligibility (“Flat Message”)
当（request/response 或 stream element）消息字段全部可映射到 C ABI 标量类型时，判定为 flat，才生成 native 变体。

支持：数字类型、bool、string、bytes。
不支持：enum、optional、repeated、map、oneof、嵌套 message。

## Native ABI Memory Rules
- 数值/bool：通过 `out` 参数返回。
- string/bytes：通过 `(outPtr, outLen, outFree)` 返回；`outPtr` 必须是可 `free` 的缓冲区，并返回可调用的 `outFree`。
- 任何 Go 分配/固定的指针输出：必须同时返回对应可调用的 free 函数指针。

## Binary ABI Encoding (protobuf bytes)
Binary 变体以 protobuf wire-format bytes 作为跨 C ABI 的请求/响应载体：
- C 调用方负责构造 request 的 protobuf bytes（serialize）。
- 生成的 Go CGO 代码 SHALL：
  - 将 request bytes `proto.Unmarshal` 为对应的 Go proto message struct，然后调用 adaptor Go API。
  - 将 adaptor 返回的 response struct `proto.Marshal` 为 bytes，并通过 `(outPtr, outLen, outFree)` 返回给 C。

因此：生成器本身不需要生成“C 侧序列化/反序列化”的代码；但 `cgotest` 的 C 测试程序必须包含 protobuf bytes 的序列化/反序列化逻辑，用于构造请求与断言响应。

## Streaming ABI
- Client-streaming 采用 staged API：Start/Send/Finish。
- stream session 使用进程内 opaque handle（建议 `uint64_t`）表示。
- Server-streaming 使用 callback：每条消息触发 onRead，结束时触发一次 onDone。
- Bidi-streaming 是 Client-streaming + Server-streaming 的结合：
  - Start：创建会话并注册 response callbacks（onRead/onDone）
  - Send：发送一个 request message（可多次调用）
  - CloseSend：关闭发送侧；接收侧继续通过 callbacks 交付直到 onDone
- Native/Binary 不混用：Start_Native 对应 Send_Native/Finish_Native。

Callbacks 约定：
- 导出函数不再接收隐式透传指针；所有回调首参为 `uint64_t call_id`，并在每次回调时显式传入。
- Binary callbacks 的 onRead 收到的是 protobuf bytes buffer + free。
- Native callbacks 的 onRead 按 response message 的字段（按 field number 升序）逐个展开参数：数值/bool 直接传值，string/bytes 以 `(ptr,len,free)` 三元组表示。

## Testing Strategy (cgotest)
- 在 `cgotest/` 下：
  - 用 `protoc` 生成 adaptor + cgo 导出代码
  - `go build -buildmode=c-shared` 生成 `.so`
  - 编写 C 测试程序调用导出 ABI，覆盖四种 RPC 类型（Binary + Native）
  - Binary 测试用例在 C 侧自行完成 protobuf bytes 的 serialize/deserialize
  - 在不同 adaptor 框架输出目录（`grpc`/`connect`/`connect_suffix`/`mix`）下重复运行
- 文档化：`cgotest/README.md` 记录运行方式；新增/补齐 build 脚本
