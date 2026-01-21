# Design: Global RPC dispatch registry

## Overview
本设计为 `rpccgo` 提供一个进程内的全局 RPC 注册与路由机制，使 adaptor 生成代码能在运行时：

1) 以 `serviceName` 注册 service handler
2) 对同一 `serviceName` 同时维护两套独立实现：`grpc` 与 `connectrpc(simple)`
3) 在调用点根据来源路由到对应 handler，并由 adaptor 负责类型断言与实际方法调用（包含 unary 与 streaming）

目标是：**把“如何找到实现并路由到正确实现”的机制稳定下来**，让生成代码专注于“把具体 RPC 方法调用写出来”。

## Key Concepts

### Two independent handler slots
同一个 `serviceName` 下允许注册两份互不影响的 handler：
- `grpc` handler：对应 gRPC 生成的 `FooServer`/`Foo_ServiceServer` 接口实现（包含 unary/streaming）。
- `connectrpc(simple)` handler：对应 connectrpc(simple option) 生成的 `FooHandler` 接口实现（通常是 unary）。

因此 registry 的逻辑键为 `(protocol, serviceName)`。

### ServiceName 与 FullMethod
- `serviceName`：形如 `rpc.bfmap.Bfmap`（示例），用于聚合一组方法。
- `fullMethod`：形如 `/rpc.bfmap.Bfmap/IsSetup`，用于在生成代码/路由层统一标识“调用的目标方法”。

选择 `/Service/Method` 形式是为了与 gRPC 生态常见约定一致，降低心智负担。

### Handler-centric registration (minimal API)
按你的反馈，本 change 的注册 API 以“注册整个 handler”为中心：
- runtime 不扫描/不反射 handler 的方法签名
- adaptor 在调用点做接口类型断言并直接调用具体方法（因此天然支持 unary 与 streaming）

该取舍的好处是：runtime API 简单、类型无关、对 streaming 不需要设计额外的通用 stream 抽象。

## Proposed runtime API (apply 阶段落地)
（以下为设计草案，proposal 阶段不写实现代码）

- 注册：
  - `RegisterGrpcHandler(serviceName string, handler any) (replaced bool, err error)`
  - `RegisterConnectHandler(serviceName string, handler any) (replaced bool, err error)`
- 查找：
  - `LookupGrpcHandler(serviceName string) (handler any, ok bool)`
  - `LookupConnectHandler(serviceName string) (handler any, ok bool)`
- 可选辅助：
  - `ListGrpcServices() []string` / `ListConnectServices() []string`（调试/可观测性）

其中：
- 默认语义为“重复注册即替换”（满足你提出的“只要重新调用注册就好”）。

## Error semantics
运行时需明确下列错误：
- handler 未注册（Lookup 返回 ok=false；或 Register/调用侧返回可判定错误）
- 注册参数非法（例如空 serviceName 或 nil handler）

设计上倾向返回可判定的 sentinel error（或带类型的 error），便于 adaptor 区分“调用失败”与“未注册”。

## Thread-safety
全局注册表必须支持并发：
- 注册通常在 init/startup 进行，但仍要防御并发调用。
- 调用路径应无锁或低开销（例如读写锁 + 读路径最小化）。

## Generator integration (adaptor)
adaptor 生成代码预期做两件事：
1) 为每个服务生成 `serviceName` 与 `fullMethod` 常量（用于路由与调试一致性）
2) 暴露注册入口：
  - `Register<Service>Grpc(handler FooServer, ...opts)`
  - `Register<Service>Connect(handler FooHandler, ...opts)`
  并在内部调用 runtime 的 `RegisterHandler(serviceName, protocol, handler, opts...)`

调用时：
- adaptor 根据调用来源（grpc vs connectrpc）选择 `protocol`
- 通过 `LookupHandler(serviceName, protocol)` 取回 handler
- 将 handler 类型断言为目标接口并调用具体方法（unary 或 streaming）

注：本 change 提案与实现仅交付“注册中心”能力；adaptor 的生成与调用路由将在后续独立 change 中完成。
