package cgotest_connect

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
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
		if err != rpcruntime.ErrServiceNotRegistered {
			t.Fatalf("expected ErrServiceNotRegistered, got %v", err)
		}
	})

	t.Run("Unary", func(t *testing.T) {
		testConnectAdaptorUnary(t)
	})

	t.Run("ClientStreaming", func(t *testing.T) {
		testConnectAdaptorClientStreaming(t)
	})

	t.Run("ServerStreaming", func(t *testing.T) {
		testConnectAdaptorServerStreaming(t)
	})

	t.Run("BidiStreaming", func(t *testing.T) {
		testConnectAdaptorBidiStreaming(t)
	})
}

func testConnectAdaptorUnary(t *testing.T) {
	// Create and register a mock handler.
	mock := &mockTestServiceHandler{}
	_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	// Call the adaptor function.
	ctx := context.Background()
	req := &PingRequest{Msg: "hello"}
	resp, err := TestService_Ping(ctx, req)
	if err != nil {
		t.Fatalf("TestService_Ping failed: %v", err)
	}

	// Verify the mock was called correctly.
	if !mock.pingCalled {
		t.Error("expected Ping to be called")
	}
	if mock.lastMsg != "hello" {
		t.Errorf("expected lastMsg to be 'hello', got %q", mock.lastMsg)
	}
	if resp.GetMsg() != "pong: hello" {
		t.Errorf("expected response 'pong: hello', got %q", resp.GetMsg())
	}
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

func testConnectAdaptorClientStreaming(t *testing.T) {
	mock := &mockStreamServiceHandlerFull{}
	_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	ctx := context.Background()

	// Start client-streaming call.
	handle, err := StreamService_ClientStreamCallStart(ctx)
	if err != nil {
		t.Fatalf("StreamService_ClientStreamCallStart failed: %v", err)
	}

	// Send messages.
	for _, msg := range []string{"A", "B", "C"} {
		if err := StreamService_ClientStreamCallSend(handle, &StreamRequest{Data: msg}); err != nil {
			t.Fatalf("StreamService_ClientStreamCallSend failed: %v", err)
		}
	}

	// Finish and get response.
	resp, err := StreamService_ClientStreamCallFinish(handle)
	if err != nil {
		t.Fatalf("StreamService_ClientStreamCallFinish failed: %v", err)
	}

	expected := "received:ABC"
	if resp.GetResult() != expected {
		t.Errorf("expected result %q, got %q", expected, resp.GetResult())
	}
}

func testConnectAdaptorServerStreaming(t *testing.T) {
	mock := &mockStreamServiceHandlerFull{}
	rpcruntime.RegisterConnectHandler(StreamService_ServiceName, mock)

	ctx := context.Background()

	var responses []string
	done := make(chan error, 1)

	onRead := func(resp *StreamResponse) bool {
		responses = append(responses, resp.GetResult())
		return true
	}

	onDone := func(err error) {
		done <- err
	}

	req := &StreamRequest{Data: "test"}
	if err := StreamService_ServerStreamCall(ctx, req, onRead, onDone); err != nil {
		t.Fatalf("StreamService_ServerStreamCall failed: %v", err)
	}

	// Wait for done callback.
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server stream failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for stream to complete")
	}

	expected := []string{"test-a", "test-b", "test-c"}
	if len(responses) != len(expected) {
		t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
	}
	for i, r := range responses {
		if r != expected[i] {
			t.Errorf("response[%d]: expected %q, got %q", i, expected[i], r)
		}
	}
}

func testConnectAdaptorBidiStreaming(t *testing.T) {
	mock := &mockStreamServiceHandlerFull{}
	rpcruntime.RegisterConnectHandler(StreamService_ServiceName, mock)

	ctx := context.Background()

	var responses []string
	done := make(chan error, 1)

	onRead := func(resp *StreamResponse) bool {
		responses = append(responses, resp.GetResult())
		return true
	}

	onDone := func(err error) {
		done <- err
	}

	// Start bidi-streaming call.
	handle, err := StreamService_BidiStreamCallStart(ctx, onRead, onDone)
	if err != nil {
		t.Fatalf("StreamService_BidiStreamCallStart failed: %v", err)
	}

	// Send messages.
	for _, msg := range []string{"X", "Y", "Z"} {
		if err := StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: msg}); err != nil {
			t.Fatalf("StreamService_BidiStreamCallSend failed: %v", err)
		}
	}

	// Close send side.
	if err := StreamService_BidiStreamCallCloseSend(handle); err != nil {
		t.Fatalf("StreamService_BidiStreamCallCloseSend failed: %v", err)
	}

	// Wait for done callback.
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bidi stream failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for stream to complete")
	}

	expected := []string{"echo:X", "echo:Y", "echo:Z"}
	if len(responses) != len(expected) {
		t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
	}
	for i, r := range responses {
		if r != expected[i] {
			t.Errorf("response[%d]: expected %q, got %q", i, expected[i], r)
		}
	}
}

// mockStreamServiceHandler is a mock implementation for stream service (uses UnimplementedStreamServiceHandler).
type mockStreamServiceHandler struct {
	UnimplementedStreamServiceHandler
}
