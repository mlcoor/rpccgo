package rpcruntime

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

var (
	connectStreamingHandlerConnType = reflect.TypeOf((*connect.StreamingHandlerConn)(nil)).Elem()

	checkConnectStreamLayoutOnce sync.Once
	checkConnectStreamLayoutErr  error
)

func checkConnFieldLayout(streamStructType reflect.Type, streamTypeName string) error {
	if streamStructType.Kind() != reflect.Struct {
		return fmt.Errorf("%s is not a struct (got %s)", streamTypeName, streamStructType.Kind())
	}
	field, ok := streamStructType.FieldByName("conn")
	if !ok {
		return fmt.Errorf("%s missing field 'conn'", streamTypeName)
	}
	if field.Offset != 0 {
		return fmt.Errorf("%s field 'conn' offset mismatch: expected 0, got %d", streamTypeName, field.Offset)
	}
	if field.Type != connectStreamingHandlerConnType {
		return fmt.Errorf(
			"%s field 'conn' type mismatch: expected %v, got %v",
			streamTypeName,
			connectStreamingHandlerConnType,
			field.Type,
		)
	}
	return nil
}

func mustCheckConnectStreamLayout() {
	checkConnectStreamLayoutOnce.Do(func() {
		// Use reflect to check that 'conn' field exists and is settable for each stream type.
		if err := checkConnFieldLayout(reflect.TypeOf(connect.ClientStream[any]{}), "connect.ClientStream[T]"); err != nil {
			checkConnectStreamLayoutErr = err
			return
		}
		if err := checkConnFieldLayout(reflect.TypeOf(connect.ServerStream[any]{}), "connect.ServerStream[T]"); err != nil {
			checkConnectStreamLayoutErr = err
			return
		}
		if err := checkConnFieldLayout(reflect.TypeOf(connect.BidiStream[any, any]{}), "connect.BidiStream[Req, Res]"); err != nil {
			checkConnectStreamLayoutErr = err
			return
		}
	})
	if checkConnectStreamLayoutErr != nil {
		panic(checkConnectStreamLayoutErr)
	}
}

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
	case req := <-c.session.SendCh():
		// Copy the received message to msg.
		// This assumes msg is a pointer to the correct type.
		if err := copyMessage(req, msg); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		return nil
	case <-c.session.SendDoneCh():
		select {
		case req := <-c.session.SendCh():
			// Copy the received message to msg.
			// This assumes msg is a pointer to the correct type.
			if err := copyMessage(req, msg); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
			return nil
		default:
			return io.EOF
		}
	case <-c.session.Context().Done():
		return c.session.Context().Err()
	}
}

// RequestHeader returns the headers received from the client.
func (c *ConnectStreamConn) RequestHeader() http.Header {
	return http.Header{}
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
func copyMessage(src, dst any) error {
	srcMsg, ok := src.(proto.Message)
	if !ok {
		return ErrStreamMessageTypeMismatch
	}
	dstMsg, ok := dst.(proto.Message)
	if !ok {
		return ErrStreamMessageTypeMismatch
	}
	if srcMsg.ProtoReflect().Descriptor().FullName() != dstMsg.ProtoReflect().Descriptor().FullName() {
		return ErrStreamMessageTypeMismatch
	}
	proto.Reset(dstMsg)
	proto.Merge(dstMsg, srcMsg)
	return nil
}

// setConnField uses reflect to set the 'conn' field of a stream struct.
// The stream must be a pointer to a struct with a 'conn' field.
func setConnField(streamPtr any, conn connect.StreamingHandlerConn, streamTypeName string) {
	rv := reflect.ValueOf(streamPtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic(fmt.Sprintf("rpcruntime: %s: expected non-nil pointer", streamTypeName))
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		panic(fmt.Sprintf("rpcruntime: %s: expected struct, got %s", streamTypeName, elem.Kind()))
	}
	connField := elem.FieldByName("conn")
	if !connField.IsValid() {
		panic(fmt.Sprintf("rpcruntime: %s: missing 'conn' field", streamTypeName))
	}
	if !connField.CanSet() {
		// Use reflect.NewAt to get a settable value for unexported field
		// This works because we're using the pointer to the struct
		connFieldAddr := reflect.NewAt(connField.Type(), connField.Addr().UnsafePointer())
		connFieldAddr.Elem().Set(reflect.ValueOf(conn))
		return
	}
	connField.Set(reflect.ValueOf(conn))
}

// SetClientStreamConn sets the conn field of a connect.ClientStream using reflect.
func SetClientStreamConn[Req any](stream *connect.ClientStream[Req], conn connect.StreamingHandlerConn) {
	if stream == nil {
		panic("rpcruntime: SetClientStreamConn called with nil stream")
	}
	mustCheckConnectStreamLayout()
	setConnField(stream, conn, "SetClientStreamConn")
}

// SetServerStreamConn sets the conn field of a connect.ServerStream using reflect.
func SetServerStreamConn[Res any](stream *connect.ServerStream[Res], conn connect.StreamingHandlerConn) {
	if stream == nil {
		panic("rpcruntime: SetServerStreamConn called with nil stream")
	}
	mustCheckConnectStreamLayout()
	setConnField(stream, conn, "SetServerStreamConn")
}

// SetBidiStreamConn sets the conn field of a connect.BidiStream using reflect.
func SetBidiStreamConn[Req, Res any](stream *connect.BidiStream[Req, Res], conn connect.StreamingHandlerConn) {
	if stream == nil {
		panic("rpcruntime: SetBidiStreamConn called with nil stream")
	}
	mustCheckConnectStreamLayout()
	setConnField(stream, conn, "SetBidiStreamConn")
}

// TrySetClientStreamConn sets the conn field of a connect.ClientStream.
//
// Unlike SetClientStreamConn, this function never panics. It returns an error
// when the stream is nil or if the underlying reflection-based injection fails.
func TrySetClientStreamConn[Req any](stream *connect.ClientStream[Req], conn connect.StreamingHandlerConn) (err error) {
	if stream == nil {
		return fmt.Errorf("rpcruntime: TrySetClientStreamConn: nil stream")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rpcruntime: TrySetClientStreamConn: %w", RecoverPanic(r))
		}
	}()
	SetClientStreamConn(stream, conn)
	return nil
}

// TrySetServerStreamConn sets the conn field of a connect.ServerStream.
//
// Unlike SetServerStreamConn, this function never panics. It returns an error
// when the stream is nil or if the underlying reflection-based injection fails.
func TrySetServerStreamConn[Res any](stream *connect.ServerStream[Res], conn connect.StreamingHandlerConn) (err error) {
	if stream == nil {
		return fmt.Errorf("rpcruntime: TrySetServerStreamConn: nil stream")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rpcruntime: TrySetServerStreamConn: %w", RecoverPanic(r))
		}
	}()
	SetServerStreamConn(stream, conn)
	return nil
}

// TrySetBidiStreamConn sets the conn field of a connect.BidiStream.
//
// Unlike SetBidiStreamConn, this function never panics. It returns an error
// when the stream is nil or if the underlying reflection-based injection fails.
func TrySetBidiStreamConn[Req, Res any](stream *connect.BidiStream[Req, Res], conn connect.StreamingHandlerConn) (err error) {
	if stream == nil {
		return fmt.Errorf("rpcruntime: TrySetBidiStreamConn: nil stream")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rpcruntime: TrySetBidiStreamConn: %w", RecoverPanic(r))
		}
	}()
	SetBidiStreamConn(stream, conn)
	return nil
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
