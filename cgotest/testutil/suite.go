package testutil

import (
	"context"
	"testing"
)

// ========== Unary RPC ==========

type RegisterFunc func() func()
type UnaryCallFunc func(ctx context.Context, msg string) (string, error)

// RunUnaryTest 执行 Unary RPC 的标准测试流程：
// 注册 handler → 调用 RPC → 验证响应匹配预期值。
func RunUnaryTest(t *testing.T, register RegisterFunc, call UnaryCallFunc, input, expected string) {
	t.Helper()
	cleanup := register()
	defer cleanup()

	ctx := context.Background()
	result, err := call(ctx, input)
	RequireNoError(t, err)
	RequireStringEqual(t, result, expected)
}

// ========== Client Streaming ==========

type ClientStreamStartFunc func(ctx context.Context) (uint64, error)
type ClientStreamSendFunc func(handle uint64, data string) error
type ClientStreamFinishFunc func(handle uint64) (string, error)

// RunClientStreamTest 执行客户端流式 RPC 的测试流程：
// 注册 handler → 开启流 → 发送多条消息 → 完成并验证响应。
func RunClientStreamTest(
	t *testing.T,
	register RegisterFunc,
	start ClientStreamStartFunc,
	send ClientStreamSendFunc,
	finish ClientStreamFinishFunc,
	inputs []string,
	expectedResult string,
) {
	t.Helper()
	cleanup := register()
	defer cleanup()

	ctx := context.Background()
	handle, err := start(ctx)
	RequireNoError(t, err)

	for _, input := range inputs {
		RequireNoError(t, send(handle, input))
	}

	result, err := finish(handle)
	RequireNoError(t, err)
	RequireStringEqual(t, result, expectedResult)
}

// ========== Server Streaming ==========

type ServerStreamCallFunc func(ctx context.Context, msg string, onRead func(string) bool) error

// RunServerStreamTest 执行服务端流式 RPC 的测试流程：
// 注册 handler → 发送请求 → 通过回调接收所有响应 → 验证响应序列。
func RunServerStreamTest(
	t *testing.T,
	register RegisterFunc,
	call ServerStreamCallFunc,
	input string,
	expectedResponses []string,
) {
	t.Helper()
	cleanup := register()
	defer cleanup()

	ctx := context.Background()
	received := make([]string, 0, len(expectedResponses))
	onRead := func(msg string) bool {
		received = append(received, msg)
		return true
	}

	RequireNoError(t, call(ctx, input, onRead))

	if len(received) != len(expectedResponses) {
		t.Fatalf("expected %d responses, got %d", len(expectedResponses), len(received))
	}
	for i := range expectedResponses {
		RequireStringEqual(t, received[i], expectedResponses[i])
	}
}

// ========== Bidirectional Streaming ==========

type BidiStreamStartFunc func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error)
type BidiStreamSendFunc func(handle uint64, data string) error
type BidiStreamCloseSendFunc func(handle uint64)

// RunBidiStreamTest 执行双向流式 RPC 的测试流程：
// 注册 handler → 开启流 → 发送消息 → 关闭发送端 → 等待完成 → 验证响应。
func RunBidiStreamTest(
	t *testing.T,
	register RegisterFunc,
	start BidiStreamStartFunc,
	send BidiStreamSendFunc,
	closeSend BidiStreamCloseSendFunc,
	inputs []string,
	expectedResponses []string,
) {
	t.Helper()
	cleanup := register()
	defer cleanup()

	ctx := context.Background()
	received := make([]string, 0, len(expectedResponses))
	done := make(chan error, 1)

	onRead := func(msg string) bool {
		received = append(received, msg)
		return true
	}
	onDone := func(err error) {
		select {
		case done <- err:
		default:
		}
	}

	handle, err := start(ctx, onRead, onDone)
	RequireNoError(t, err)

	for _, input := range inputs {
		RequireNoError(t, send(handle, input))
	}

	closeSend(handle)

	RequireNoError(t, <-done)

	if len(received) != len(expectedResponses) {
		t.Fatalf("expected %d responses, got %d", len(expectedResponses), len(received))
	}
	for i := range expectedResponses {
		RequireStringEqual(t, received[i], expectedResponses[i])
	}
}
