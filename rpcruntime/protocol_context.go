package rpcruntime

import "context"

// contextKey 是 context 携带 protocol 的非导出 key 类型，避免冲突。
type contextKey struct{}

// ContextKeyProtocol 是用于在 context 中携带 protocol 选择的 key。
// 使用者可通过 WithProtocol 和 ProtocolFromContext 来操作。
var ContextKeyProtocol = contextKey{}

// WithProtocol 返回携带 protocol 选择的新 context。
//
// 示例:
//
//	ctx := rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)
func WithProtocol(ctx context.Context, protocol Protocol) context.Context {
	return context.WithValue(ctx, ContextKeyProtocol, protocol)
}

// ProtocolFromContext 从 context 提取 protocol 选择。
//
// 如果 context 中未设置 protocol，返回空字符串和 false。
func ProtocolFromContext(ctx context.Context) (Protocol, bool) {
	protocol, ok := ctx.Value(ContextKeyProtocol).(Protocol)
	return protocol, ok
}
