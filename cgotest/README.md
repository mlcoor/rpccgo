# cgotest

这个目录用于验证 `protoc-gen-rpc-cgo` 生成的 CGO C ABI 是否能在真实 C 程序里端到端调用（构建 `.so` + C 代码链接运行）。

## 前置条件

- `protoc`
- C 编译器：`cc`/`gcc`/`clang`
- Go 工具链

脚本会自动安装（从当前 workspace）:
- `protoc-gen-rpc-cgo-adaptor`
- `protoc-gen-rpc-cgo`

## go-task 安装与使用

### 安装 go-task

    go install github.com/go-task/task/v3/cmd/task@latest

### 基本用法

查看可用任务：

    task --list

运行完整测试套件：

    task test

运行特定协议的 Go adaptor 测试：

    task adaptor-test PROTOCOL=grpc
    task adaptor-test PROTOCOL=connect
    task adaptor-test PROTOCOL=connect_suffix
    task adaptor-test PROTOCOL=mix

运行特定协议的 C 端到端测试：

    task c-test PROTOCOL=grpc
    task c-test PROTOCOL=connect
    task c-test PROTOCOL=connect_suffix
    task c-test PROTOCOL=mix

构建特定协议的共享库：

    task build PROTOCOL=grpc
    task build PROTOCOL=connect
    task build PROTOCOL=connect_suffix
    task build PROTOCOL=mix

清理生成的文件和构建产物：

    task clean

重新生成所有代码（pb + adaptor + C ABI）：

    task regen

安装 protoc 插件：

    task install-plugins

### 注意事项

如果直接运行 `task` 而不指定任务名称，会显示错误。
请使用 `task --list` 查看所有可用任务，然后明确指定要运行的任务。

## 一键运行（推荐）

在仓库根目录执行：

- `./cgotest/test.sh`

它会：
- 生成各协议矩阵的 Go 代码 + adaptor
- 跑 Go 单测（用于验证 adaptor 行为）
- 生成 rpc-cgo 导出代码，构建 `c_tests/libygrpc.so` + `c_tests/libygrpc.h`
- 编译并运行 C 端到端测试程序

## 只跑某个协议的 C 端到端

在 `cgotest/` 目录执行：

- `./run-c-tests.sh connect`
- `./run-c-tests.sh grpc`
- `./run-c-tests.sh connect_suffix`
- `./run-c-tests.sh mix`

## C 测试覆盖面

`c_tests/` 下的程序覆盖：
- Unary：Binary + Native（含 TakeReq/free 策略验证）
- Client-streaming：Binary + Native（含 TakeReq/free 策略验证）
- Server-streaming：Binary + Native（callback + resp_free 验证）
- Bidi-streaming：Binary + Native（Start 注册回调、Send、CloseSend）

补充说明：协议选择不再读取环境变量 `YGRPC_PROTOCOL`。

- 如需在 C 侧强制指定协议，请在发起 RPC 前调用 `Ygrpc_SetProtocol(...)`：
	- `YGRPC_PROTOCOL_UNSET`：清除默认协议
	- `YGRPC_PROTOCOL_GRPC`
	- `YGRPC_PROTOCOL_CONNECTRPC`
