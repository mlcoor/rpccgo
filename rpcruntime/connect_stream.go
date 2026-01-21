package rpcruntime

import (
	"io"
	"net/http"
	"unsafe"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

// ConnectStreamConn implements connect.StreamingHandlerConn for CGO adaptor use.
// This bridges rpcruntime.StreamSession with Connect's streaming expectations.
type ConnectStreamConn struct {
	session StreamSession
}

// NewConnectStreamConn creates a new ConnectStreamConn.
func NewConnectStreamConn(session StreamSession) *ConnectStreamConn {
	return &ConnectStreamConn{session: session}
}

// Spec returns the specification for the RPC.
func (c *ConnectStreamConn) Spec() connect.Spec {
	return connect.Spec{}
}

// Peer describes the client for this RPC.
func (c *ConnectStreamConn) Peer() connect.Peer {
	return connect.Peer{}
}

// Receive reads a message from the session's send channel.
func (c *ConnectStreamConn) Receive(msg any) error {
	select {
	case req, ok := <-c.session.SendCh():
		if !ok {
			// Channel closed means end of stream.
			return io.EOF
		}
		// Copy the received message to msg.
		// This assumes msg is a pointer to the correct type.
		copyMessage(req, msg)
		return nil
	case <-c.session.Context().Done():
		return c.session.Context().Err()
	}
}

// RequestHeader returns the headers received from the client.
func (c *ConnectStreamConn) RequestHeader() http.Header {
	return nil
}

// Send sends a message to the client via the onRead callback.
func (c *ConnectStreamConn) Send(msg any) error {
	if cb := c.session.OnRead(); cb != nil {
		if !cb(msg) {
			return connect.NewError(connect.CodeCanceled, nil)
		}
	}
	return nil
}

// ResponseHeader returns the response headers.
func (c *ConnectStreamConn) ResponseHeader() http.Header {
	return nil
}

// ResponseTrailer returns the response trailers.
func (c *ConnectStreamConn) ResponseTrailer() http.Header {
	return nil
}

// copyMessage copies src to dst using proto.Merge.
// Both src and dst should be pointers to the same proto message type.
func copyMessage(src, dst any) {
	srcMsg, ok := src.(proto.Message)
	if !ok {
		return
	}
	dstMsg, ok := dst.(proto.Message)
	if !ok {
		return
	}
	proto.Merge(dstMsg, srcMsg)
}

// clientStreamFields mirrors the internal layout of connect.ClientStream[T].
// WARNING: This relies on the internal structure of connect.ClientStream and may break
// if the connect library changes its internal layout.
type clientStreamFields struct {
	conn        connect.StreamingHandlerConn
	initializer any
	msg         any
	err         error
}

// serverStreamFields mirrors the internal layout of connect.ServerStream[T].
type serverStreamFields struct {
	conn connect.StreamingHandlerConn
}

// bidiStreamFields mirrors the internal layout of connect.BidiStream[Req, Res].
type bidiStreamFields struct {
	conn        connect.StreamingHandlerConn
	initializer any
}

// SetClientStreamConn sets the conn field of a connect.ClientStream using unsafe.
func SetClientStreamConn[Req any](stream *connect.ClientStream[Req], conn connect.StreamingHandlerConn) {
	fields := (*clientStreamFields)(unsafe.Pointer(stream))
	fields.conn = conn
}

// SetServerStreamConn sets the conn field of a connect.ServerStream using unsafe.
func SetServerStreamConn[Res any](stream *connect.ServerStream[Res], conn connect.StreamingHandlerConn) {
	fields := (*serverStreamFields)(unsafe.Pointer(stream))
	fields.conn = conn
}

// SetBidiStreamConn sets the conn field of a connect.BidiStream using unsafe.
func SetBidiStreamConn[Req, Res any](stream *connect.BidiStream[Req, Res], conn connect.StreamingHandlerConn) {
	fields := (*bidiStreamFields)(unsafe.Pointer(stream))
	fields.conn = conn
}

// NewClientStream creates a new connect.ClientStream with the given conn.
func NewClientStream[Req any](conn connect.StreamingHandlerConn) *connect.ClientStream[Req] {
	stream := &connect.ClientStream[Req]{}
	SetClientStreamConn(stream, conn)
	return stream
}

// NewServerStream creates a new connect.ServerStream with the given conn.
func NewServerStream[Res any](conn connect.StreamingHandlerConn) *connect.ServerStream[Res] {
	stream := &connect.ServerStream[Res]{}
	SetServerStreamConn(stream, conn)
	return stream
}

// NewBidiStream creates a new connect.BidiStream with the given conn.
func NewBidiStream[Req, Res any](conn connect.StreamingHandlerConn) *connect.BidiStream[Req, Res] {
	stream := &connect.BidiStream[Req, Res]{}
	SetBidiStreamConn(stream, conn)
	return stream
}
