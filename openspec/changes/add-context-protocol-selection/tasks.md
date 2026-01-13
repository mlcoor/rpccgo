## 1. Specs
- [x] 1.1 更新 `rpc-cgo-adaptor` spec delta：支持多 protocol 参数 + 通用 adaptor 的选择语义
- [x] 1.2 更新 `rpc-dispatch` spec delta：新增/统一 protocol 选择的 context key 与 helper API
- [x] 1.3 运行 `openspec validate add-context-protocol-selection --strict`

## 2. Implementation（批准后执行）

- [x] 2.1 扩展插件参数解析：新增 `protocol`（逗号分隔、有序列表，默认 `connectrpc`）
- [x] 2.1.1 扩展插件参数解析：新增可选 `connect_package_suffix`（默认空字符串；非空时 connect handler interface 位于 `<current-import-path>/<current-go-package-name><suffix>`）
- [x] 2.2 生成通用 adaptor `*_cgo_adaptor.go`：按 `protocol` 选项生成不同的 dispatch 逻辑（单协议仅查该协议；多协议支持 ctx 指定与 fallback）
- [x] 2.3 更新 `cgotest/` 下的 build 脚本：把 `--rpc-cgo-adaptor_opt=framework=...` 迁移为 `--rpc-cgo-adaptor_opt=protocol=...`
- [x] 2.4 在 `rpcruntime` 增加 protocol 选择相关 context key + helper functions
- [x] 2.5 在 `cgotest/all/` 增加混合注册测试：验证 ctx 指定与 fallback 行为
	- 覆盖：ctx 指定 grpc 但仅注册 connectrpc → 期望错误（不 fallback）
	- 覆盖：ctx 不指定 + 注册 connectrpc → 期望 fallback 命中 connectrpc
	- 覆盖：单协议 `protocol=grpc` + 注册 connectrpc → 期望错误（不 lookup connectrpc）
- [x] 2.5.1 `cgotest/all/` 生成策略：base 包生成 messages + go-grpc + adaptor；connect-go 生成到独立子包（默认 package_suffix），并通过 adaptor 选项 `connect_package_suffix` 指向该子包
- [x] 2.5.2 新增 `cgotest/connect_suffix/` 测试：connect-go 使用非空 package_suffix 输出到独立子包，adaptor 使用 `connect_package_suffix` 验证可调用
- [x] 2.6 在 `cgotest/` 增加 `build-all.sh`：一键生成 grpc/connectrpc/all 三套测试生成物（用于 CI/本地快速验证）
- [x] 2.7 运行 `go test ./...`
