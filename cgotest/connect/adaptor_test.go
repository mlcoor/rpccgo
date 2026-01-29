package cgotest_connect

import (
	"context"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

// mockTestServiceHandler is a mock implementation of TestServiceHandler for testing.
type mockTestServiceHandler struct {
	UnimplementedTestServiceHandler
	pingCalled bool
	lastMsg    string
}

func (m *mockTestServiceHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	m.pingCalled = true
	m.lastMsg = req.GetMsg()
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func TestConnectAdaptor(t *testing.T) {
	t.Run("ServiceNotRegistered", func(t *testing.T) {
		ctx := context.Background()
		req := &PingRequest{Msg: "test"}

		_, err := TestService_Ping(ctx, req)
		testutil.RequireEqual(t, err, rpcruntime.ErrServiceNotRegistered)
	})

	t.Run("Unary", func(t *testing.T) {
		mock := &mockTestServiceHandler{}
		testutil.RunUnaryTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
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

		if !mock.pingCalled {
			t.Error("expected Ping to be called")
		}
		if mock.lastMsg != "hello" {
			t.Errorf("expected lastMsg to be 'hello', got %q", mock.lastMsg)
		}
	})

	t.Run("ClientStreaming", func(t *testing.T) {
		testutil.RunClientStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, &mockStreamServiceHandlerFull{})
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
				_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, &mockStreamServiceHandlerFull{})
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

				select {
				case err := <-done:
					return err
				case <-time.After(5 * time.Second):
					return context.DeadlineExceeded
				}
			},
			"test",
			[]string{"test-a", "test-b", "test-c"},
		)
	})

	t.Run("BidiStreaming", func(t *testing.T) {
		testutil.RunBidiStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, &mockStreamServiceHandlerFull{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
				wrappedDone, startTimer := wrapOnDoneWithTimeout(onDone)
				handle, err := StreamService_BidiStreamCallStart(
					ctx,
					func(resp *StreamResponse) bool {
						return onRead(resp.GetResult())
					},
					wrappedDone,
				)
				if err != nil {
					return 0, err
				}
				startTimer()
				return handle, nil
			},
			func(handle uint64, data string) error {
				return StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: data})
			},
			func(handle uint64) {
				if err := StreamService_BidiStreamCallCloseSend(handle); err != nil {
					t.Fatalf("StreamService_BidiStreamCallCloseSend failed: %v", err)
				}
			},
			[]string{"X", "Y", "Z"},
			[]string{"echo:X", "echo:Y", "echo:Z"},
		)
	})
}

func wrapOnDoneWithTimeout(onDone func(error)) (func(error), func()) {
	done := make(chan struct{})
	var once sync.Once
	wrapped := func(err error) {
		once.Do(func() {
			close(done)
			onDone(err)
		})
	}
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

// Test Connect streaming methods.

// mockStreamServiceHandlerFull implements StreamServiceHandler with working streaming.
type mockStreamServiceHandlerFull struct {
	UnimplementedStreamServiceHandler
	clientStreamMsgs []string
}

func (m *mockStreamServiceHandlerFull) ClientStreamCall(
	ctx context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	m.clientStreamMsgs = nil
	for stream.Receive() {
		m.clientStreamMsgs = append(m.clientStreamMsgs, stream.Msg().GetData())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	total := ""
	for _, msg := range m.clientStreamMsgs {
		total += msg
	}
	return &StreamResponse{Result: "received:" + total}, nil
}

func (m *mockStreamServiceHandlerFull) ServerStreamCall(
	ctx context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockStreamServiceHandlerFull) BidiStreamCall(
	ctx context.Context,
	stream *connect.BidiStream[StreamRequest, StreamResponse],
) error {
	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		resp := &StreamResponse{Result: "echo:" + req.GetData()}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

// mockStreamServiceHandler is a mock implementation for stream service (uses UnimplementedStreamServiceHandler).
type mockStreamServiceHandler struct {
	UnimplementedStreamServiceHandler
}
