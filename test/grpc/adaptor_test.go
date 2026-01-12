package cgotest

import (
	"context"
	"testing"

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


func TestGrpcAdaptor(t *testing.T) {
	t.Run("ServiceNotRegistered", func(t *testing.T) {
		ctx := context.Background()
		req := &PingRequest{Msg: "test"}

		_, err := TestService_Ping(ctx, req)
		if err != rpcruntime.ErrServiceNotRegistered {
			t.Fatalf("expected ErrServiceNotRegistered, got %v", err)
		}
	})

	t.Run("Unary", func(t *testing.T) {
		testGrpcAdaptorUnary(t)
	})

	t.Run("ClientStreaming", func(t *testing.T) {
		testGrpcAdaptorClientStreaming(t)
	})

	t.Run("ServerStreaming", func(t *testing.T) {
		testGrpcAdaptorServerStreaming(t)
	})

	t.Run("BidiStreaming", func(t *testing.T) {
		testGrpcAdaptorBidiStreaming(t)
	})
}

func testGrpcAdaptorUnary(t *testing.T) {
	// Create and register a mock handler.
	mock := &mockTestServiceServer{}
	_, err := rpcruntime.RegisterGrpcHandler(TestService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterGrpcHandler failed: %v", err)
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

func testGrpcAdaptorClientStreaming(t *testing.T) {
	mock := &mockStreamServiceServer{}
	_, err := rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterGrpcHandler failed: %v", err)
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

func testGrpcAdaptorServerStreaming(t *testing.T) {
	mock := &mockStreamServiceServer{}
	rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, mock)

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
	if err := <-done; err != nil {
		t.Fatalf("server stream failed: %v", err)
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

func testGrpcAdaptorBidiStreaming(t *testing.T) {
	mock := &mockStreamServiceServer{}
	rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, mock)

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
	for _, msg := range []string{"A", "B", "C"} {
		if err := StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: msg}); err != nil {
			t.Fatalf("StreamService_BidiStreamCallSend failed: %v", err)
		}
	}

	// Close send side.
	if err := StreamService_BidiStreamCallCloseSend(handle); err != nil {
		t.Fatalf("StreamService_BidiStreamCallCloseSend failed: %v", err)
	}

	// Wait for done callback.
	if err := <-done; err != nil {
		t.Fatalf("bidi stream failed: %v", err)
	}

	expected := []string{"echo:A", "echo:B", "echo:C"}
	if len(responses) != len(expected) {
		t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
	}
	for i, r := range responses {
		if r != expected[i] {
			t.Errorf("response[%d]: expected %q, got %q", i, expected[i], r)
		}
	}
}
