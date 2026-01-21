package main

import (
	"context"

	"connectrpc.com/connect"
	cgotest_connect "github.com/ygrpc/rpccgo/cgotest/connect"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type testServiceConnect struct {
	cgotest_connect.UnimplementedTestServiceHandler
}

func (s *testServiceConnect) Ping(ctx context.Context, req *cgotest_connect.PingRequest) (*cgotest_connect.PingResponse, error) {
	return &cgotest_connect.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceConnect) PingOpt1(ctx context.Context, req *cgotest_connect.PingRequestOpt1) (*cgotest_connect.PingResponse, error) {
	return &cgotest_connect.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceConnect) PingOpt2(ctx context.Context, req *cgotest_connect.PingRequestOpt2) (*cgotest_connect.PingResponse, error) {
	return &cgotest_connect.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type streamServiceConnect struct {
	cgotest_connect.UnimplementedStreamServiceHandler
}

func (s *streamServiceConnect) UnaryCall(ctx context.Context, req *cgotest_connect.StreamRequest) (*cgotest_connect.StreamResponse, error) {
	return &cgotest_connect.StreamResponse{Result: "ok:" + req.GetData(), Sequence: req.GetSequence()}, nil
}

func (s *streamServiceConnect) ClientStreamCall(ctx context.Context, stream *connect.ClientStream[cgotest_connect.StreamRequest]) (*cgotest_connect.StreamResponse, error) {
	total := ""
	var lastSeq int32
	for stream.Receive() {
		msg := stream.Msg()
		total += msg.GetData()
		lastSeq = msg.GetSequence()
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &cgotest_connect.StreamResponse{Result: "received:" + total, Sequence: lastSeq}, nil
}

func (s *streamServiceConnect) ServerStreamCall(ctx context.Context, req *cgotest_connect.StreamRequest, stream *connect.ServerStream[cgotest_connect.StreamResponse]) error {
	for i := 0; i < 3; i++ {
		resp := &cgotest_connect.StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i)), Sequence: int32(i)}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamServiceConnect) BidiStreamCall(ctx context.Context, stream *connect.BidiStream[cgotest_connect.StreamRequest, cgotest_connect.StreamResponse]) error {
	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		resp := &cgotest_connect.StreamResponse{Result: "echo:" + req.GetData(), Sequence: req.GetSequence()}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	_, _ = rpcruntime.RegisterConnectHandler(cgotest_connect.TestService_ServiceName, &testServiceConnect{})
	_, _ = rpcruntime.RegisterConnectHandler(cgotest_connect.StreamService_ServiceName, &streamServiceConnect{})
}
