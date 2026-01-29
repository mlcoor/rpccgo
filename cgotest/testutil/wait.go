package testutil

import (
	"testing"
	"time"
)

// WaitDone waits for a channel to receive a value or times out.
// If the channel receives an error, it returns the error.
// If timeout occurs, it calls t.Fatal.
func WaitDone(t *testing.T, done chan error, timeout time.Duration) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		t.Fatal("timeout waiting for operation to complete")
		return nil
	}
}
