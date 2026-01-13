## MODIFIED Requirements

### Requirement: Protocols selection via plugin options
protoc 插件 SHALL 支持一个选项，用于控制生成哪些 protocol 对应的 adaptor 代码。

该选项的值 SHALL 为以 `;` 分隔的有序 protocol 标识符列表：
- `grpc`
- `connectrpc`

如果该选项被省略，默认值 SHALL 为 `connectrpc`。

#### Scenario: Default protocol list is connectrpc
- **GIVEN** a proto input with at least one service
- **WHEN** `protoc-gen-rpc-cgo-adaptor` runs with no protocol selection option
- **THEN** it SHALL behave as if the protocol list were `connectrpc`

#### Scenario: Protocol list supports multiple entries
- **GIVEN** a proto input with at least one service
- **WHEN** `protoc-gen-rpc-cgo-adaptor` runs with `protocol=grpc|connectrpc`
- **THEN** it SHALL generate adaptor code supporting both `grpc` and `connectrpc`

---

### Requirement: Connect handler package is configurable
当生成 `connectrpc` 相关 adaptor 代码时，如果 connect 的 service/handler 接口类型不在当前 Go package 内（例如 `protoc-gen-connect-go` 生成到独立 package，或配置了非空 `package_suffix`），插件 SHALL 支持一个可选参数用于指定 connect-go 的 `package_suffix`，以推导 connect handler interface 的 Go import path。

- Option name: `connect_package_suffix`
- Default value is empty string.
- If empty, the adaptor SHALL assume the connect handler interface type is in the current Go package.
- If non-empty, the adaptor SHALL assume the connect handler interface type is in the connect-go generated sub-package whose import path is `<current-import-path>/<current-go-package-name><connect_package_suffix>`.

Notes:
- `current-go-package-name` refers to the Go package name of the generated adaptor file (i.e. the base package where `*.pb.go` is generated).
- A concrete derivation that matches `protoc-gen-connect-go` behavior is:
	- `connectSubpackageName = <current-go-package-name> + <connect_package_suffix>`
	- `connectHandlerImportPath = <current-import-path> + "/" + connectSubpackageName`

#### Scenario: connect_package_suffix enables connect-go suffix packages
- **GIVEN** `protoc-gen-connect-go` is configured to generate connect code into a separate Go package
- **AND** `protoc-gen-rpc-cgo-adaptor` is invoked with `protocol=connectrpc`
- **WHEN** `protoc-gen-rpc-cgo-adaptor` is invoked with `connect_package_suffix=connect`
- **THEN** the generated adaptor code SHALL reference the connect handler interface type from `<current-import-path>/<current-go-package-name>connect`

---

### Requirement: Dispatch via global registry using protocol selection
在运行时，生成的 adaptor 函数 SHALL 根据传入的 `ctx` 中携带的 `protocol` 值来选择 dispatch lookup 路径。

系统 SHALL 将 `protocol` 选择视为 `rpcruntime.Protocol` 标识符。

如果 `ctx` 未携带 protocol 值，adaptor SHALL 按插件选项配置的顺序依次尝试 protocol dispatch，直到找到 handler。

如果 `ctx` 携带了 protocol 值，adaptor SHALL 仅使用该 protocol 进行 dispatch。

#### Scenario: Explicit protocol dispatches to grpc handler
- **GIVEN** `ctx` carries `protocol = rpcruntime.ProtocolGrpc`
- **AND** a grpc handler is registered for `serviceName`
- **WHEN** the generated adaptor function is invoked
- **THEN** it SHALL lookup via `LookupGrpcHandler`
- **AND** call the grpc service method implementation

#### Scenario: Explicit protocol dispatches to connect handler
- **GIVEN** `ctx` carries `protocol = rpcruntime.ProtocolConnectRPC`
- **AND** a connectrpc handler is registered for `serviceName`
- **WHEN** the generated adaptor function is invoked
- **THEN** it SHALL lookup via `LookupConnectHandler`
- **AND** call the connectrpc service method implementation

#### Scenario: Missing protocol falls back across configured list
- **GIVEN** `ctx` does not carry a protocol value
- **AND** the generated adaptor is configured with `protocol=grpc|connectrpc`
- **AND** a connectrpc handler is registered for `serviceName`
- **WHEN** the generated adaptor function is invoked
- **THEN** it SHALL attempt grpc lookup first
- **AND** it SHALL then attempt connectrpc lookup
- **AND** it SHALL call the connectrpc service method implementation

#### Scenario: Single protocol list attempts only that protocol
- **GIVEN** `ctx` does not carry a protocol value
- **AND** the generated adaptor is configured with `protocol=grpc`
- **AND** no grpc handler is registered for `serviceName`
- **AND** a connectrpc handler is registered for `serviceName`
- **WHEN** the generated adaptor function is invoked
- **THEN** it SHALL attempt grpc lookup
- **AND** it SHALL NOT attempt connectrpc lookup
- **AND** it SHALL return a non-nil error

---

### Requirement: Deterministic errors for routing failures
生成的 adaptor 代码 SHALL 至少对以下情况返回确定性的错误：
- `ctx` carries an unknown/unsupported protocol value
- No handler is registered for the selected `(protocol, serviceName)`
- No handler is registered for any configured protocol (fallback)
- Registered handler has an unexpected type (type assertion fails)

#### Scenario: Unknown protocol returns error
- **GIVEN** `ctx` carries a protocol value that is not supported by the generated adaptor
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

#### Scenario: Explicit protocol not registered returns error
- **GIVEN** `ctx` carries `protocol = rpcruntime.ProtocolGrpc`
- **AND** no grpc handler is registered for `serviceName`
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

#### Scenario: Fallback finds no handlers returns error
- **GIVEN** `ctx` does not carry a protocol value
- **AND** no handler is registered for any configured protocol for `serviceName`
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error

#### Scenario: Type mismatch returns error
- **GIVEN** a handler is registered for the selected `(protocol, serviceName)`
- **AND** the handler cannot be asserted to the expected service interface
- **WHEN** the adaptor function is called
- **THEN** it SHALL return a non-nil error
