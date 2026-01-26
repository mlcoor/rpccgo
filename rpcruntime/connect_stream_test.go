package rpcruntime

import (
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// getConnField uses reflect to read the 'conn' field value from a stream struct.
func getConnField(streamPtr any) connect.StreamingHandlerConn {
	rv := reflect.ValueOf(streamPtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return nil
	}
	connField := elem.FieldByName("conn")
	if !connField.IsValid() {
		return nil
	}
	// Use reflect.NewAt to read unexported field
	connFieldPtr := reflect.NewAt(connField.Type(), connField.Addr().UnsafePointer())
	connValue := connFieldPtr.Elem().Interface()
	if conn, ok := connValue.(connect.StreamingHandlerConn); ok {
		return conn
	}
	return nil
}

func TestConnectStreamStructLayout(t *testing.T) {
	t.Run("ReflectLayout", func(t *testing.T) {
		if err := checkConnFieldLayout(reflect.TypeOf(connect.ClientStream[any]{}), "connect.ClientStream[T]"); err != nil {
			t.Fatalf("client stream layout check failed: %v", err)
		}
		if err := checkConnFieldLayout(reflect.TypeOf(connect.ServerStream[any]{}), "connect.ServerStream[T]"); err != nil {
			t.Fatalf("server stream layout check failed: %v", err)
		}
		if err := checkConnFieldLayout(reflect.TypeOf(connect.BidiStream[any, any]{}), "connect.BidiStream[Req, Res]"); err != nil {
			t.Fatalf("bidi stream layout check failed: %v", err)
		}
	})

	t.Run("ReflectInitialNil", func(t *testing.T) {
		client := &connect.ClientStream[any]{}
		if getConnField(client) != nil {
			t.Fatal("expected client conn to be nil initially")
		}

		server := &connect.ServerStream[any]{}
		if getConnField(server) != nil {
			t.Fatal("expected server conn to be nil initially")
		}

		bidi := &connect.BidiStream[any, any]{}
		if getConnField(bidi) != nil {
			t.Fatal("expected bidi conn to be nil initially")
		}
	})
}

func TestCheckConnFieldLayoutErrors(t *testing.T) {
	type wrongOffset struct {
		x    int
		conn connect.StreamingHandlerConn
	}
	type wrongType struct {
		conn int
	}
	type missingConn struct {
		Conn connect.StreamingHandlerConn
	}

	if err := checkConnFieldLayout(reflect.TypeOf(wrongOffset{}), "wrongOffset"); err == nil {
		t.Fatal("expected error for wrongOffset, got nil")
	}
	if err := checkConnFieldLayout(reflect.TypeOf(wrongType{}), "wrongType"); err == nil {
		t.Fatal("expected error for wrongType, got nil")
	}
	if err := checkConnFieldLayout(reflect.TypeOf(missingConn{}), "missingConn"); err == nil {
		t.Fatal("expected error for missingConn, got nil")
	}
}

// TestSetStreamConn verifies that SetXxxStreamConn functions work correctly.
func TestSetStreamConn(t *testing.T) {
	session := &streamSession{}
	conn := NewConnectStreamConn(session)

	t.Run("SetClientStreamConn", func(t *testing.T) {
		stream := &connect.ClientStream[any]{}
		SetClientStreamConn(stream, conn)

		if getConnField(stream) != conn {
			t.Error("SetClientStreamConn did not set conn correctly")
		}
	})

	t.Run("SetServerStreamConn", func(t *testing.T) {
		stream := &connect.ServerStream[any]{}
		SetServerStreamConn(stream, conn)

		if getConnField(stream) != conn {
			t.Error("SetServerStreamConn did not set conn correctly")
		}
	})

	t.Run("SetBidiStreamConn", func(t *testing.T) {
		stream := &connect.BidiStream[any, any]{}
		SetBidiStreamConn(stream, conn)

		if getConnField(stream) != conn {
			t.Error("SetBidiStreamConn did not set conn correctly")
		}
	})
}

func TestTrySetStreamConn(t *testing.T) {
	session := &streamSession{}
	conn := NewConnectStreamConn(session)

	t.Run("NilReturnsError", func(t *testing.T) {
		if err := TrySetClientStreamConn[any](nil, conn); err == nil {
			t.Fatal("expected error for nil client stream, got nil")
		}
		if err := TrySetServerStreamConn[any](nil, conn); err == nil {
			t.Fatal("expected error for nil server stream, got nil")
		}
		if err := TrySetBidiStreamConn[any, any](nil, conn); err == nil {
			t.Fatal("expected error for nil bidi stream, got nil")
		}
	})

	t.Run("SetsConnSuccessfully", func(t *testing.T) {
		client := &connect.ClientStream[any]{}
		if err := TrySetClientStreamConn(client, conn); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if getConnField(client) != conn {
			t.Fatal("TrySetClientStreamConn did not set conn correctly")
		}

		server := &connect.ServerStream[any]{}
		if err := TrySetServerStreamConn(server, conn); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if getConnField(server) != conn {
			t.Fatal("TrySetServerStreamConn did not set conn correctly")
		}

		bidi := &connect.BidiStream[any, any]{}
		if err := TrySetBidiStreamConn(bidi, conn); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if getConnField(bidi) != conn {
			t.Fatal("TrySetBidiStreamConn did not set conn correctly")
		}
	})
}

func TestNewStreamsSetConn(t *testing.T) {
	session := &streamSession{}
	conn := NewConnectStreamConn(session)

	client := NewClientStream[any](conn)
	if getConnField(client) != conn {
		t.Fatal("NewClientStream did not set conn correctly")
	}

	server := NewServerStream[any](conn)
	if getConnField(server) != conn {
		t.Fatal("NewServerStream did not set conn correctly")
	}

	bidi := NewBidiStream[any, any](conn)
	if getConnField(bidi) != conn {
		t.Fatal("NewBidiStream did not set conn correctly")
	}
}

func TestCopyMessage(t *testing.T) {
	t.Run("TypeMatchCopies", func(t *testing.T) {
		src := &wrapperspb.StringValue{Value: "hello"}
		dst := &wrapperspb.StringValue{Value: "stale"}

		if err := copyMessage(src, dst); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dst.Value != "hello" {
			t.Fatalf("expected dst to be updated, got %q", dst.Value)
		}
	})

	t.Run("TypeMismatchReturnsError", func(t *testing.T) {
		src := &wrapperspb.StringValue{Value: "hello"}
		dst := &wrapperspb.Int32Value{Value: 1}

		if err := copyMessage(src, dst); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonProtoReturnsError", func(t *testing.T) {
		if err := copyMessage("not proto", "also not proto"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
