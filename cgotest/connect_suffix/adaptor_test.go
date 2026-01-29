package cgotest_connect_suffix

import (
	"connectrpc.com/connect"
	"context"
	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockConnectSuffixHandler struct{}

func (m *mockConnectSuffixHandler) Ping(_ context.Context, req *PingRequest) (*PingResponse, error) {
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type mockConnectSuffixStreamServiceHandler struct{}

func (m *mockConnectSuffixStreamServiceHandler) ClientStreamCall(
	ctx context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	_ = ctx
	var builder strings.Builder
	for stream.Receive() {
		builder.WriteString(stream.Msg().GetData())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &StreamResponse{Result: "received:" + builder.String()}, nil
}

func (m *mockConnectSuffixStreamServiceHandler) ServerStreamCall(
	ctx context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	_ = ctx
	prefix := req.GetData() + "-"
	if err := stream.Send(&StreamResponse{Result: prefix + "a"}); err != nil {
		return err
	}
	if err := stream.Send(&StreamResponse{Result: prefix + "b"}); err != nil {
		return err
	}
	return stream.Send(&StreamResponse{Result: prefix + "c"})
}

func (m *mockConnectSuffixStreamServiceHandler) BidiStreamCall(
	ctx context.Context,
	stream *connect.BidiStream[StreamRequest, StreamResponse],
) error {
	_ = ctx
	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

func registerConnect(t *testing.T, name string, handler any) testutil.RegisterFunc {
	return func() func() {
		_, err := rpcruntime.RegisterConnectHandler(name, handler)
		testutil.RequireNoError(t, err)
		return func() {}
	}
}

func wrapOnDoneWithTimeout(onDone func(error)) (func(error), func()) {
	done := make(chan struct{})
	var once sync.Once
	wrapped := func(err error) { once.Do(func() { close(done); onDone(err) }) }
	startTimer := func() {
		go func() {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				wrapped(context.DeadlineExceeded)
			}
		}()
	}
	return wrapped, startTimer
}

func TestConnectSuffixAdaptor(t *testing.T) {
	t.Run("NoProtocol_DefaultConnect", func(t *testing.T) {
		testutil.RunUnaryTest(t, registerConnect(t, TestService_ServiceName, &mockConnectSuffixHandler{}), func(ctx context.Context, msg string) (string, error) {
			resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
			if err != nil {
				return "", err
			}
			return resp.GetMsg(), nil
		}, "hello", "pong: hello")
	})
	t.Run("ExplicitConnectRPC", func(t *testing.T) {
		testutil.RunUnaryTest(t, registerConnect(t, TestService_ServiceName, &mockConnectSuffixHandler{}), func(ctx context.Context, msg string) (string, error) {
			ctx = rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolConnectRPC)
			resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
			if err != nil {
				return "", err
			}
			return resp.GetMsg(), nil
		}, "hello", "pong: hello")
	})
}

func TestConnectSuffixAdaptor_StreamServiceStreaming(t *testing.T) {
	t.Run("ClientStreaming", func(t *testing.T) {
		testutil.RunClientStreamTest(t, registerConnect(t, StreamService_ServiceName, &mockConnectSuffixStreamServiceHandler{}), func(ctx context.Context) (uint64, error) { return StreamService_ClientStreamCallStart(ctx) }, func(handle uint64, data string) error {
			return StreamService_ClientStreamCallSend(handle, &StreamRequest{Data: data})
		}, func(handle uint64) (string, error) {
			resp, err := StreamService_ClientStreamCallFinish(handle)
			if err != nil {
				return "", err
			}
			return resp.GetResult(), nil
		}, []string{"A", "B", "C"}, "received:ABC")
	})
	t.Run("ServerStreaming", func(t *testing.T) {
		testutil.RunServerStreamTest(t, registerConnect(t, StreamService_ServiceName, &mockConnectSuffixStreamServiceHandler{}), func(ctx context.Context, msg string, onRead func(string) bool) error {
			done := make(chan error, 1)
			onDone := func(err error) { done <- err }
			if err := StreamService_ServerStreamCall(ctx, &StreamRequest{Data: msg}, func(resp *StreamResponse) bool { return onRead(resp.GetResult()) }, onDone); err != nil {
				return err
			}
			select {
			case err := <-done:
				return err
			case <-time.After(5 * time.Second):
				return context.DeadlineExceeded
			}
		}, "test", []string{"test-a", "test-b", "test-c"})
	})
	t.Run("BidiStreaming", func(t *testing.T) {
		testutil.RunBidiStreamTest(t, registerConnect(t, StreamService_ServiceName, &mockConnectSuffixStreamServiceHandler{}), func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
			wrappedDone, startTimer := wrapOnDoneWithTimeout(onDone)
			handle, err := StreamService_BidiStreamCallStart(ctx, func(resp *StreamResponse) bool { return onRead(resp.GetResult()) }, wrappedDone)
			if err != nil {
				return 0, err
			}
			startTimer()
			return handle, nil
		}, func(handle uint64, data string) error {
			return StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: data})
		}, func(handle uint64) { testutil.RequireNoError(t, StreamService_BidiStreamCallCloseSend(handle)) }, []string{"X", "Y", "Z"}, []string{"echo:X", "echo:Y", "echo:Z"})
	})
}
