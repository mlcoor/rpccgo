package cgotest_grpc

import (
	"context"
	"strings"
	"testing"

	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockTestServiceServer struct {
	UnimplementedTestServiceServer
}

func (m *mockTestServiceServer) Ping(_ context.Context, req *PingRequest) (*PingResponse, error) {
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type mockConnectHandler struct{ called *bool }

func (m *mockConnectHandler) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	if m.called != nil {
		*m.called = true
	}
	return &PingResponse{Msg: "should-not-happen"}, nil
}

type mockStreamServiceServer struct {
	UnimplementedStreamServiceServer
}

func (m *mockStreamServiceServer) ClientStreamCall(stream StreamService_ClientStreamCallServer) error {
	var builder strings.Builder
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		builder.WriteString(req.GetData())
	}
	return stream.SendAndClose(&StreamResponse{Result: "received:" + builder.String()})
}

func (m *mockStreamServiceServer) ServerStreamCall(
	req *StreamRequest,
	stream StreamService_ServerStreamCallServer,
) error {
	prefix := req.GetData() + "-"
	if err := stream.Send(&StreamResponse{Result: prefix + "a"}); err != nil {
		return err
	}
	if err := stream.Send(&StreamResponse{Result: prefix + "b"}); err != nil {
		return err
	}
	return stream.Send(&StreamResponse{Result: prefix + "c"})
}

func (m *mockStreamServiceServer) BidiStreamCall(stream StreamService_BidiStreamCallServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

func registerGrpc(t *testing.T, name string, handler any) testutil.RegisterFunc {
	return func() func() {
		_, err := rpcruntime.RegisterGrpcHandler(name, handler)
		testutil.RequireNoError(t, err)
		return func() {}
	}
}

func TestGrpcAdaptor(t *testing.T) {
	t.Run("ServiceNotRegistered", func(t *testing.T) {
		_, err := TestService_Ping(context.Background(), &PingRequest{Msg: "test"})
		testutil.RequireEqual(t, err, rpcruntime.ErrServiceNotRegistered)
	})
	t.Run("SingleProtocolGrpc_IgnoresConnectHandler", func(t *testing.T) {
		connectCalled := false
		_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, &mockConnectHandler{called: &connectCalled})
		testutil.RequireNoError(t, err)
		_, callErr := TestService_Ping(context.Background(), &PingRequest{Msg: "hello"})
		testutil.RequireEqual(t, callErr, rpcruntime.ErrServiceNotRegistered)
		if connectCalled {
			t.Fatalf("expected connect handler not to be called")
		}
	})
	t.Run("Unary", func(t *testing.T) {
		testutil.RunUnaryTest(t, registerGrpc(t, TestService_ServiceName, &mockTestServiceServer{}), func(ctx context.Context, msg string) (string, error) {
			resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
			if err != nil {
				return "", err
			}
			return resp.GetMsg(), nil
		}, "hello", "pong: hello")
	})
	t.Run("ClientStreaming", func(t *testing.T) {
		testutil.RunClientStreamTest(t, registerGrpc(t, StreamService_ServiceName, &mockStreamServiceServer{}), func(ctx context.Context) (uint64, error) { return StreamService_ClientStreamCallStart(ctx) }, func(handle uint64, data string) error {
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
		testutil.RunServerStreamTest(t, registerGrpc(t, StreamService_ServiceName, &mockStreamServiceServer{}), func(ctx context.Context, msg string, onRead func(string) bool) error {
			done := make(chan error, 1)
			onDone := func(err error) { done <- err }
			if err := StreamService_ServerStreamCall(ctx, &StreamRequest{Data: msg}, func(resp *StreamResponse) bool { return onRead(resp.GetResult()) }, onDone); err != nil {
				return err
			}
			return <-done
		}, "test", []string{"test-a", "test-b", "test-c"})
	})
	t.Run("BidiStreaming", func(t *testing.T) {
		testutil.RunBidiStreamTest(t, registerGrpc(t, StreamService_ServiceName, &mockStreamServiceServer{}), func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
			return StreamService_BidiStreamCallStart(ctx, func(resp *StreamResponse) bool { return onRead(resp.GetResult()) }, onDone)
		}, func(handle uint64, data string) error {
			return StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: data})
		}, func(handle uint64) { testutil.RequireNoError(t, StreamService_BidiStreamCallCloseSend(handle)) }, []string{"A", "B", "C"}, []string{"echo:A", "echo:B", "echo:C"})
	})
}
