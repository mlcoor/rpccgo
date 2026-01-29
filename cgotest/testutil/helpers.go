// Package testutil provides testing helper functions for rpccgo tests.
package testutil

import "testing"

// RequireNoError asserts that the error is nil.
// If the error is not nil, it calls t.Fatal with the error message.
func RequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// RequireError asserts that the error is not nil.
// If the error is nil, it calls t.Fatal.
func RequireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
}

// RequireEqual asserts that two comparable values are equal.
// If they are not equal, it calls t.Error with both values.
func RequireEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// RequireStringEqual asserts that two strings are equal.
// If they are not equal, it calls t.Error with both strings.
func RequireStringEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
