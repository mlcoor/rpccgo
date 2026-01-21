package main

import (
	"context"
	"io"

	"connectrpc.com/connect"
	cgotest_mix "github.com/ygrpc/rpccgo/cgotest/mix"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

// ConnectRPC handlers (mix).

type testServiceMixConnect struct{}

type streamServiceMixConnect struct{}

func (s *testServiceMixConnect) Ping(ctx context.Context, req *cgotest_mix.PingRequest) (*cgotest_mix.PingResponse, error) {
	return &cgotest_mix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceMixConnect) PingOpt1(ctx context.Context, req *cgotest_mix.PingRequestOpt1) (*cgotest_mix.PingResponse, error) {
	return &cgotest_mix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceMixConnect) PingOpt2(ctx context.Context, req *cgotest_mix.PingRequestOpt2) (*cgotest_mix.PingResponse, error) {
	return &cgotest_mix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *streamServiceMixConnect) UnaryCall(ctx context.Context, req *cgotest_mix.StreamRequest) (*cgotest_mix.StreamResponse, error) {
	return &cgotest_mix.StreamResponse{Result: "ok:" + req.GetData(), Sequence: req.GetSequence()}, nil
}

func (s *streamServiceMixConnect) ClientStreamCall(ctx context.Context, stream *connect.ClientStream[cgotest_mix.StreamRequest]) (*cgotest_mix.StreamResponse, error) {
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
	return &cgotest_mix.StreamResponse{Result: "received:" + total, Sequence: lastSeq}, nil
}

func (s *streamServiceMixConnect) ServerStreamCall(ctx context.Context, req *cgotest_mix.StreamRequest, stream *connect.ServerStream[cgotest_mix.StreamResponse]) error {
	for i := 0; i < 3; i++ {
		resp := &cgotest_mix.StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i)), Sequence: int32(i)}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamServiceMixConnect) BidiStreamCall(ctx context.Context, stream *connect.BidiStream[cgotest_mix.StreamRequest, cgotest_mix.StreamResponse]) error {
	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		resp := &cgotest_mix.StreamResponse{Result: "echo:" + req.GetData(), Sequence: req.GetSequence()}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

// gRPC handlers (mix).

type testServiceMixGrpc struct{ cgotest_mix.UnimplementedTestServiceServer }

type streamServiceMixGrpc struct{ cgotest_mix.UnimplementedStreamServiceServer }

func (s *testServiceMixGrpc) Ping(ctx context.Context, req *cgotest_mix.PingRequest) (*cgotest_mix.PingResponse, error) {
	return &cgotest_mix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceMixGrpc) PingOpt1(ctx context.Context, req *cgotest_mix.PingRequestOpt1) (*cgotest_mix.PingResponse, error) {
	return &cgotest_mix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceMixGrpc) PingOpt2(ctx context.Context, req *cgotest_mix.PingRequestOpt2) (*cgotest_mix.PingResponse, error) {
	return &cgotest_mix.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *streamServiceMixGrpc) UnaryCall(ctx context.Context, req *cgotest_mix.StreamRequest) (*cgotest_mix.StreamResponse, error) {
	return &cgotest_mix.StreamResponse{Result: "ok:" + req.GetData(), Sequence: req.GetSequence()}, nil
}

func (s *streamServiceMixGrpc) ClientStreamCall(stream cgotest_mix.StreamService_ClientStreamCallServer) error {
	total := ""
	var lastSeq int32
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		total += req.GetData()
		lastSeq = req.GetSequence()
	}
	return stream.SendAndClose(&cgotest_mix.StreamResponse{Result: "received:" + total, Sequence: lastSeq})
}

func (s *streamServiceMixGrpc) ServerStreamCall(req *cgotest_mix.StreamRequest, stream cgotest_mix.StreamService_ServerStreamCallServer) error {
	for i := 0; i < 3; i++ {
		resp := &cgotest_mix.StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i)), Sequence: int32(i)}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamServiceMixGrpc) BidiStreamCall(stream cgotest_mix.StreamService_BidiStreamCallServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		resp := &cgotest_mix.StreamResponse{Result: "echo:" + req.GetData(), Sequence: req.GetSequence()}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	_, _ = rpcruntime.RegisterConnectHandler(cgotest_mix.TestService_ServiceName, &testServiceMixConnect{})
	_, _ = rpcruntime.RegisterConnectHandler(cgotest_mix.StreamService_ServiceName, &streamServiceMixConnect{})

	_, _ = rpcruntime.RegisterGrpcHandler(cgotest_mix.TestService_ServiceName, &testServiceMixGrpc{})
	_, _ = rpcruntime.RegisterGrpcHandler(cgotest_mix.StreamService_ServiceName, &streamServiceMixGrpc{})
}
