package rpcruntime

import (
	"context"
	"testing"
)

func TestWithProtocolAndProtocolFromContext(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
	}{
		{"grpc", ProtocolGrpc},
		{"connectrpc", ProtocolConnectRPC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithProtocol(ctx, tt.protocol)

			got, ok := ProtocolFromContext(ctx)
			if !ok {
				t.Fatal("expected ok=true")
			}
			if got != tt.protocol {
				t.Errorf("expected %q, got %q", tt.protocol, got)
			}
		})
	}
}

func TestProtocolFromContext_NotSet(t *testing.T) {
	ctx := context.Background()

	got, ok := ProtocolFromContext(ctx)
	if ok {
		t.Error("expected ok=false when protocol not set")
	}
	if got != "" {
		t.Errorf("expected empty protocol, got %q", got)
	}
}

func TestProtocolFromContext_Overwrite(t *testing.T) {
	ctx := context.Background()

	ctx = WithProtocol(ctx, ProtocolGrpc)
	ctx = WithProtocol(ctx, ProtocolConnectRPC)

	got, ok := ProtocolFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != ProtocolConnectRPC {
		t.Errorf("expected %q after overwrite, got %q", ProtocolConnectRPC, got)
	}
}
