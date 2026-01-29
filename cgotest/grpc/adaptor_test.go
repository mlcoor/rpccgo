package cgotest_grpc

import (
	"context"
	"testing"

	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

// mockTestServiceServer is a mock implementation of TestServiceServer for testing.
type mockTestServiceServer struct {
	UnimplementedTestServiceServer
	pingCalled bool
	lastMsg    string
}

func (m *mockTestServiceServer) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	m.pingCalled = true
	m.lastMsg = req.GetMsg()
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type mockConnectHandler struct {
	called *bool
}

func (m *mockConnectHandler) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	if m.called != nil {
		*m.called = true
	}
	return &PingResponse{Msg: "should-not-happen"}, nil
}

func TestGrpcAdaptor(t *testing.T) {
	t.Run("ServiceNotRegistered", func(t *testing.T) {
		ctx := context.Background()
		req := &PingRequest{Msg: "test"}

		_, err := TestService_Ping(ctx, req)
		testutil.RequireEqual(t, err, rpcruntime.ErrServiceNotRegistered)
	})

	t.Run("SingleProtocolGrpc_IgnoresConnectHandler", func(t *testing.T) {
		connectCalled := false
		_, err := rpcruntime.RegisterConnectHandler(
			TestService_ServiceName,
			&mockConnectHandler{called: &connectCalled},
		)
		testutil.RequireNoError(t, err)

		_, callErr := TestService_Ping(context.Background(), &PingRequest{Msg: "hello"})
		testutil.RequireEqual(t, callErr, rpcruntime.ErrServiceNotRegistered)
		if connectCalled {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("Unary", func(t *testing.T) {
		testutil.RunUnaryTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterGrpcHandler(TestService_ServiceName, &mockTestServiceServer{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, msg string) (string, error) {
				resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
				if err != nil {
					return "", err
				}
				return resp.GetMsg(), nil
			},
			"hello",
			"pong: hello",
		)
	})

	t.Run("ClientStreaming", func(t *testing.T) {
		testutil.RunClientStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, &mockStreamServiceServer{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context) (uint64, error) {
				return StreamService_ClientStreamCallStart(ctx)
			},
			func(handle uint64, data string) error {
				return StreamService_ClientStreamCallSend(handle, &StreamRequest{Data: data})
			},
			func(handle uint64) (string, error) {
				resp, err := StreamService_ClientStreamCallFinish(handle)
				if err != nil {
					return "", err
				}
				return resp.GetResult(), nil
			},
			[]string{"A", "B", "C"},
			"received:ABC",
		)
	})

	t.Run("ServerStreaming", func(t *testing.T) {
		testutil.RunServerStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, &mockStreamServiceServer{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, msg string, onRead func(string) bool) error {
				done := make(chan error, 1)
				onDone := func(err error) {
					done <- err
				}
				if err := StreamService_ServerStreamCall(
					ctx,
					&StreamRequest{Data: msg},
					func(resp *StreamResponse) bool {
						return onRead(resp.GetResult())
					},
					onDone,
				); err != nil {
					return err
				}
				return <-done
			},
			"test",
			[]string{"test-a", "test-b", "test-c"},
		)
	})

	t.Run("BidiStreaming", func(t *testing.T) {
		testutil.RunBidiStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, &mockStreamServiceServer{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
				return StreamService_BidiStreamCallStart(
					ctx,
					func(resp *StreamResponse) bool {
						return onRead(resp.GetResult())
					},
					onDone,
				)
			},
			func(handle uint64, data string) error {
				return StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: data})
			},
			func(handle uint64) {
				if err := StreamService_BidiStreamCallCloseSend(handle); err != nil {
					t.Fatalf("StreamService_BidiStreamCallCloseSend failed: %v", err)
				}
			},
			[]string{"A", "B", "C"},
			[]string{"echo:A", "echo:B", "echo:C"},
		)
	})
}

// mockStreamServiceServer is a mock implementation for streaming tests.
type mockStreamServiceServer struct {
	UnimplementedStreamServiceServer
	clientStreamMsgs []string
	serverStreamResp []*StreamResponse
}

func (m *mockStreamServiceServer) ClientStreamCall(stream StreamService_ClientStreamCallServer) error {
	m.clientStreamMsgs = nil
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		m.clientStreamMsgs = append(m.clientStreamMsgs, req.GetData())
	}
	total := ""
	for _, msg := range m.clientStreamMsgs {
		total += msg
	}
	return stream.SendAndClose(&StreamResponse{Result: "received:" + total})
}

func (m *mockStreamServiceServer) ServerStreamCall(
	req *StreamRequest,
	stream StreamService_ServerStreamCallServer,
) error {
	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockStreamServiceServer) BidiStreamCall(stream StreamService_BidiStreamCallServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		// Echo back with prefix.
		if err := stream.Send(&StreamResponse{Result: "echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}
