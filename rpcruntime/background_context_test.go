package rpcruntime

import (
	"testing"
)

func TestBackgroundContext_DefaultDoesNotSetProtocol(t *testing.T) {
	ClearDefaultProtocol()

	ctx := BackgroundContext()
	got, ok := ProtocolFromContext(ctx)
	if ok {
		t.Fatalf("expected ok=false, got ok=true with %q", got)
	}
}

func TestBackgroundContext_UsesDefaultProtocol(t *testing.T) {
	if err := SetDefaultProtocol(ProtocolGrpc); err != nil {
		t.Fatalf("SetDefaultProtocol: %v", err)
	}
	t.Cleanup(ClearDefaultProtocol)

	ctx := BackgroundContext()
	got, ok := ProtocolFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != ProtocolGrpc {
		t.Fatalf("expected %q, got %q", ProtocolGrpc, got)
	}
}
