package main

import (
	"context"

	"connectrpc.com/connect"
	cgotest_connect_suffix "github.com/ygrpc/rpccgo/cgotest/connect_suffix"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type testServiceConnectSuffix struct{}

type streamServiceConnectSuffix struct{}

func (s *testServiceConnectSuffix) Ping(ctx context.Context, req *cgotest_connect_suffix.PingRequest) (*cgotest_connect_suffix.PingResponse, error) {
	return &cgotest_connect_suffix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceConnectSuffix) PingOpt1(ctx context.Context, req *cgotest_connect_suffix.PingRequestOpt1) (*cgotest_connect_suffix.PingResponse, error) {
	return &cgotest_connect_suffix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceConnectSuffix) PingOpt2(ctx context.Context, req *cgotest_connect_suffix.PingRequestOpt2) (*cgotest_connect_suffix.PingResponse, error) {
	return &cgotest_connect_suffix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *streamServiceConnectSuffix) UnaryCall(ctx context.Context, req *cgotest_connect_suffix.StreamRequest) (*cgotest_connect_suffix.StreamResponse, error) {
	return &cgotest_connect_suffix.StreamResponse{Result: "ok:" + req.GetData(), Sequence: req.GetSequence()}, nil
}

func (s *streamServiceConnectSuffix) ClientStreamCall(ctx context.Context, stream *connect.ClientStream[cgotest_connect_suffix.StreamRequest]) (*cgotest_connect_suffix.StreamResponse, error) {
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
	return &cgotest_connect_suffix.StreamResponse{Result: "received:" + total, Sequence: lastSeq}, nil
}

func (s *streamServiceConnectSuffix) ServerStreamCall(ctx context.Context, req *cgotest_connect_suffix.StreamRequest, stream *connect.ServerStream[cgotest_connect_suffix.StreamResponse]) error {
	for i := 0; i < 3; i++ {
		resp := &cgotest_connect_suffix.StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i)), Sequence: int32(i)}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamServiceConnectSuffix) BidiStreamCall(ctx context.Context, stream *connect.BidiStream[cgotest_connect_suffix.StreamRequest, cgotest_connect_suffix.StreamResponse]) error {
	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		resp := &cgotest_connect_suffix.StreamResponse{Result: "echo:" + req.GetData(), Sequence: req.GetSequence()}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	_, _ = rpcruntime.RegisterConnectHandler(cgotest_connect_suffix.TestService_ServiceName, &testServiceConnectSuffix{})
	_, _ = rpcruntime.RegisterConnectHandler(cgotest_connect_suffix.StreamService_ServiceName, &streamServiceConnectSuffix{})
}
