package testutil

import (
	"errors"
	"testing"

	"github.com/ygrpc/rpccgo/rpcruntime"
)

// RequireServiceNotRegisteredError asserts that the error is ErrServiceNotRegistered.
// If the error does not match, it calls t.Fatal with the actual error.
func RequireServiceNotRegisteredError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, rpcruntime.ErrServiceNotRegistered) {
		t.Fatalf("expected ErrServiceNotRegistered, got: %v", err)
	}
}
