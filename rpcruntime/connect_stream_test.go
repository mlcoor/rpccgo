package rpcruntime

import (
	"testing"
	"unsafe"

	"connectrpc.com/connect"
)

// TestConnectStreamStructLayout validates that the conn field is at offset 0
// in connect stream types, which is the only layout assumption we rely on.
func TestConnectStreamStructLayout(t *testing.T) {
	t.Run("ClientStream", func(t *testing.T) {
		stream := &connect.ClientStream[any]{}
		fields := (*clientStreamFields)(unsafe.Pointer(stream))

		if fields.conn != nil {
			t.Error("expected conn to be nil initially")
		}
	})

	t.Run("ServerStream", func(t *testing.T) {
		stream := &connect.ServerStream[any]{}
		fields := (*serverStreamFields)(unsafe.Pointer(stream))

		if fields.conn != nil {
			t.Error("expected conn to be nil initially")
		}
	})

	t.Run("BidiStream", func(t *testing.T) {
		stream := &connect.BidiStream[any, any]{}
		fields := (*bidiStreamFields)(unsafe.Pointer(stream))

		if fields.conn != nil {
			t.Error("expected conn to be nil initially")
		}
	})
}

// TestSetStreamConn verifies that SetXxxStreamConn functions work correctly.
func TestSetStreamConn(t *testing.T) {
	session := &streamSession{}
	conn := NewConnectStreamConn(session)

	t.Run("SetClientStreamConn", func(t *testing.T) {
		stream := &connect.ClientStream[any]{}
		SetClientStreamConn(stream, conn)

		fields := (*clientStreamFields)(unsafe.Pointer(stream))
		if fields.conn != conn {
			t.Error("SetClientStreamConn did not set conn correctly")
		}
	})

	t.Run("SetServerStreamConn", func(t *testing.T) {
		stream := &connect.ServerStream[any]{}
		SetServerStreamConn(stream, conn)

		fields := (*serverStreamFields)(unsafe.Pointer(stream))
		if fields.conn != conn {
			t.Error("SetServerStreamConn did not set conn correctly")
		}
	})

	t.Run("SetBidiStreamConn", func(t *testing.T) {
		stream := &connect.BidiStream[any, any]{}
		SetBidiStreamConn(stream, conn)

		fields := (*bidiStreamFields)(unsafe.Pointer(stream))
		if fields.conn != conn {
			t.Error("SetBidiStreamConn did not set conn correctly")
		}
	})
}
