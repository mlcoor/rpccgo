package main

import (
	"context"
	"io"

	cgotest_grpc "github.com/ygrpc/rpccgo/cgotest/grpc"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type testServiceGrpc struct {
	cgotest_grpc.UnimplementedTestServiceServer
}

func (s *testServiceGrpc) Ping(ctx context.Context, req *cgotest_grpc.PingRequest) (*cgotest_grpc.PingResponse, error) {
	return &cgotest_grpc.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceGrpc) PingOpt1(
	ctx context.Context,
	req *cgotest_grpc.PingRequestOpt1,
) (*cgotest_grpc.PingResponse, error) {
	return &cgotest_grpc.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func (s *testServiceGrpc) PingOpt2(
	ctx context.Context,
	req *cgotest_grpc.PingRequestOpt2,
) (*cgotest_grpc.PingResponse, error) {
	return &cgotest_grpc.PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type streamServiceGrpc struct {
	cgotest_grpc.UnimplementedStreamServiceServer
}

func (s *streamServiceGrpc) UnaryCall(
	ctx context.Context,
	req *cgotest_grpc.StreamRequest,
) (*cgotest_grpc.StreamResponse, error) {
	return &cgotest_grpc.StreamResponse{Result: "ok:" + req.GetData(), Sequence: req.GetSequence()}, nil
}

func (s *streamServiceGrpc) ClientStreamCall(stream cgotest_grpc.StreamService_ClientStreamCallServer) error {
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
	return stream.SendAndClose(&cgotest_grpc.StreamResponse{Result: "received:" + total, Sequence: lastSeq})
}

func (s *streamServiceGrpc) ServerStreamCall(
	req *cgotest_grpc.StreamRequest,
	stream cgotest_grpc.StreamService_ServerStreamCallServer,
) error {
	for i := 0; i < 3; i++ {
		resp := &cgotest_grpc.StreamResponse{Result: req.GetData() + "-" + string(rune('a'+i)), Sequence: int32(i)}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamServiceGrpc) BidiStreamCall(stream cgotest_grpc.StreamService_BidiStreamCallServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		resp := &cgotest_grpc.StreamResponse{Result: "echo:" + req.GetData(), Sequence: req.GetSequence()}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	_, _ = rpcruntime.RegisterGrpcHandler(cgotest_grpc.TestService_ServiceName, &testServiceGrpc{})
	_, _ = rpcruntime.RegisterGrpcHandler(cgotest_grpc.StreamService_ServiceName, &streamServiceGrpc{})
}
