package cgotest_mix

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockConnectTestServiceHandler struct {
	pingCalled bool
	lastMsg    string
}

func (m *mockConnectTestServiceHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	m.pingCalled = true
	m.lastMsg = req.GetMsg()
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func TestAllAdaptor_ContextSelection(t *testing.T) {
	t.Run("ExplicitGrpc_NoFallback", func(t *testing.T) {
		mock := &mockConnectTestServiceHandler{}
		_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
		if err != nil {
			t.Fatalf("RegisterConnectHandler failed: %v", err)
		}

		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolGrpc)
		_, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		if callErr != rpcruntime.ErrServiceNotRegistered {
			t.Fatalf("expected ErrServiceNotRegistered, got %v", callErr)
		}
		if mock.pingCalled {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("NoProtocol_FallbackToConnect", func(t *testing.T) {
		mock := &mockConnectTestServiceHandler{}
		_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
		if err != nil {
			t.Fatalf("RegisterConnectHandler failed: %v", err)
		}

		ctx := context.Background()
		resp, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		if callErr != nil {
			t.Fatalf("TestService_Ping failed: %v", callErr)
		}
		if !mock.pingCalled {
			t.Fatalf("expected connect handler to be called")
		}
		if mock.lastMsg != "hello" {
			t.Fatalf("expected lastMsg to be 'hello', got %q", mock.lastMsg)
		}
		if resp.GetMsg() != "pong: hello" {
			t.Fatalf("expected response 'pong: hello', got %q", resp.GetMsg())
		}
	})
}

type mockMixConnectStreamServiceHandler struct {
	clientStreamCalled int32
	serverStreamCalled int32
	bidiStreamCalled   int32
}

func (m *mockMixConnectStreamServiceHandler) ClientStreamCall(
	ctx context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	_ = ctx
	atomic.AddInt32(&m.clientStreamCalled, 1)

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
	return &StreamResponse{Result: "connect:received:" + joined}, nil
}

func (m *mockMixConnectStreamServiceHandler) ServerStreamCall(
	ctx context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	_ = ctx
	atomic.AddInt32(&m.serverStreamCalled, 1)

	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: "connect:" + req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockMixConnectStreamServiceHandler) BidiStreamCall(
	ctx context.Context,
	stream *connect.BidiStream[StreamRequest, StreamResponse],
) error {
	_ = ctx
	atomic.AddInt32(&m.bidiStreamCalled, 1)

	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "connect:echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

type mockMixGrpcStreamServiceServer struct {
	UnimplementedStreamServiceServer
	clientStreamCalled int32
	serverStreamCalled int32
	bidiStreamCalled   int32
}

func (m *mockMixGrpcStreamServiceServer) ClientStreamCall(stream StreamService_ClientStreamCallServer) error {
	atomic.AddInt32(&m.clientStreamCalled, 1)

	var msgs []string
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		msgs = append(msgs, req.GetData())
	}
	joined := ""
	for _, s := range msgs {
		joined += s
	}
	return stream.SendAndClose(&StreamResponse{Result: "grpc:received:" + joined})
}

func (m *mockMixGrpcStreamServiceServer) ServerStreamCall(
	req *StreamRequest,
	stream StreamService_ServerStreamCallServer,
) error {
	atomic.AddInt32(&m.serverStreamCalled, 1)

	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: "grpc:" + req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockMixGrpcStreamServiceServer) BidiStreamCall(stream StreamService_BidiStreamCallServer) error {
	atomic.AddInt32(&m.bidiStreamCalled, 1)

	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "grpc:echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

func TestMixAdaptor_StreamServiceStreaming(t *testing.T) {
	connectMock := &mockMixConnectStreamServiceHandler{}
	_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, connectMock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	t.Run("NoProtocol_FallbackToConnect", func(t *testing.T) {
		ctx := context.Background()

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
		if resp.GetResult() != "connect:received:ABC" {
			t.Fatalf("expected %q, got %q", "connect:received:ABC", resp.GetResult())
		}

		var responses []string
		done := make(chan error, 1)
		onRead := func(resp *StreamResponse) bool {
			responses = append(responses, resp.GetResult())
			return true
		}
		onDone := func(err error) {
			done <- err
		}
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
		expected := []string{"connect:test-a", "connect:test-b", "connect:test-c"}
		if len(responses) != len(expected) {
			t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
		}
		for i := range expected {
			if responses[i] != expected[i] {
				t.Fatalf("response[%d]: expected %q, got %q", i, expected[i], responses[i])
			}
		}

		responses = nil
		done = make(chan error, 1)
		onRead = func(resp *StreamResponse) bool {
			responses = append(responses, resp.GetResult())
			return true
		}
		onDone = func(err error) {
			done <- err
		}
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
		expected = []string{"connect:echo:X", "connect:echo:Y", "connect:echo:Z"}
		if len(responses) != len(expected) {
			t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
		}
		for i := range expected {
			if responses[i] != expected[i] {
				t.Fatalf("response[%d]: expected %q, got %q", i, expected[i], responses[i])
			}
		}
	})

	t.Run("ExplicitGrpc_NoFallback", func(t *testing.T) {
		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolGrpc)
		before := atomic.LoadInt32(&connectMock.serverStreamCalled)

		done := make(chan error, 1)
		onRead := func(*StreamResponse) bool { return true }
		onDone := func(err error) { done <- err }
		err := StreamService_ServerStreamCall(ctx, &StreamRequest{Data: "test"}, onRead, onDone)
		if err != rpcruntime.ErrServiceNotRegistered {
			t.Fatalf("expected ErrServiceNotRegistered, got %v", err)
		}
		if after := atomic.LoadInt32(&connectMock.serverStreamCalled); after != before {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("ExplicitGrpc_UsesGrpc", func(t *testing.T) {
		grpcMock := &mockMixGrpcStreamServiceServer{}
		_, err := rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, grpcMock)
		if err != nil {
			t.Fatalf("RegisterGrpcHandler failed: %v", err)
		}

		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolGrpc)

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
		if resp.GetResult() != "grpc:received:ABC" {
			t.Fatalf("expected %q, got %q", "grpc:received:ABC", resp.GetResult())
		}

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
		expected := []string{"grpc:test-a", "grpc:test-b", "grpc:test-c"}
		if len(responses) != len(expected) {
			t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
		}
		for i := range expected {
			if responses[i] != expected[i] {
				t.Fatalf("response[%d]: expected %q, got %q", i, expected[i], responses[i])
			}
		}

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
		expected = []string{"grpc:echo:X", "grpc:echo:Y", "grpc:echo:Z"}
		if len(responses) != len(expected) {
			t.Fatalf("expected %d responses, got %d", len(expected), len(responses))
		}
		for i := range expected {
			if responses[i] != expected[i] {
				t.Fatalf("response[%d]: expected %q, got %q", i, expected[i], responses[i])
			}
		}
	})
}
