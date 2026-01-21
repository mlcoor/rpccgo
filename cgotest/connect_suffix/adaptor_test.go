package cgotest_connect_suffix

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockConnectSuffixHandler struct {
	pingCalled bool
	lastMsg    string
}

func (m *mockConnectSuffixHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	m.pingCalled = true
	m.lastMsg = req.GetMsg()
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func TestConnectSuffixAdaptor(t *testing.T) {
	mock := &mockConnectSuffixHandler{}
	_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	t.Run("NoProtocol_DefaultConnect", func(t *testing.T) {
		resp, callErr := TestService_Ping(context.Background(), &PingRequest{Msg: "hello"})
		if callErr != nil {
			t.Fatalf("TestService_Ping failed: %v", callErr)
		}
		if !mock.pingCalled {
			t.Fatalf("expected handler to be called")
		}
		if resp.GetMsg() != "pong: hello" {
			t.Fatalf("expected response 'pong: hello', got %q", resp.GetMsg())
		}
	})

	t.Run("ExplicitConnectRPC", func(t *testing.T) {
		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolConnectRPC)
		resp, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		if callErr != nil {
			t.Fatalf("TestService_Ping failed: %v", callErr)
		}
		if resp.GetMsg() != "pong: hello" {
			t.Fatalf("expected response 'pong: hello', got %q", resp.GetMsg())
		}
	})
}

type mockConnectSuffixStreamServiceHandler struct{}

func (m *mockConnectSuffixStreamServiceHandler) ClientStreamCall(
	ctx context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	_ = ctx
	var msgs []string
	for stream.Receive() {
		msgs = append(msgs, stream.Msg().GetData())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	joined := ""
	for _, s := range msgs {
		joined += s
	}
	return &StreamResponse{Result: "received:" + joined}, nil
}

func (m *mockConnectSuffixStreamServiceHandler) ServerStreamCall(
	ctx context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	_ = ctx
	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
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

func TestConnectSuffixAdaptor_StreamServiceStreaming(t *testing.T) {
	mock := &mockConnectSuffixStreamServiceHandler{}
	_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	ctx := context.Background()

	// Client streaming
	handle, err := StreamService_ClientStreamCallStart(ctx)
	if err != nil {
		t.Fatalf("StreamService_ClientStreamCallStart failed: %v", err)
	}
	for _, msg := range []string{"A", "B", "C"} {
		if err := StreamService_ClientStreamCallSend(handle, &StreamRequest{Data: msg}); err != nil {
			t.Fatalf("StreamService_ClientStreamCallSend failed: %v", err)
		}
	}
	resp, err := StreamService_ClientStreamCallFinish(handle)
	if err != nil {
		t.Fatalf("StreamService_ClientStreamCallFinish failed: %v", err)
	}
	if resp.GetResult() != "received:ABC" {
		t.Fatalf("expected %q, got %q", "received:ABC", resp.GetResult())
	}

	// Server streaming
	var responses []string
	done := make(chan error, 1)
	onRead := func(resp *StreamResponse) bool {
		responses = append(responses, resp.GetResult())
		return true
	}
	onDone := func(err error) { done <- err }
	if err := StreamService_ServerStreamCall(ctx, &StreamRequest{Data: "test"}, onRead, onDone); err != nil {
		t.Fatalf("StreamService_ServerStreamCall failed: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server stream failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server stream to complete")
	}
	expected := []string{"test-a", "test-b", "test-c"}
	if len(responses) != len(expected) {
		t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
	}
	for i := range expected {
		if responses[i] != expected[i] {
			t.Fatalf("response[%d]: expected %q, got %q", i, expected[i], responses[i])
		}
	}

	// Bidi streaming
	responses = nil
	done = make(chan error, 1)
	onRead = func(resp *StreamResponse) bool {
		responses = append(responses, resp.GetResult())
		return true
	}
	onDone = func(err error) { done <- err }
	handle, err = StreamService_BidiStreamCallStart(ctx, onRead, onDone)
	if err != nil {
		t.Fatalf("StreamService_BidiStreamCallStart failed: %v", err)
	}
	for _, msg := range []string{"X", "Y", "Z"} {
		if err := StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: msg}); err != nil {
			t.Fatalf("StreamService_BidiStreamCallSend failed: %v", err)
		}
	}
	if err := StreamService_BidiStreamCallCloseSend(handle); err != nil {
		t.Fatalf("StreamService_BidiStreamCallCloseSend failed: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bidi stream failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for bidi stream to complete")
	}
	expected = []string{"echo:X", "echo:Y", "echo:Z"}
	if len(responses) != len(expected) {
		t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
	}
	for i := range expected {
		if responses[i] != expected[i] {
			t.Fatalf("response[%d]: expected %q, got %q", i, expected[i], responses[i])
		}
	}
}
