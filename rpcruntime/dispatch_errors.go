package rpcruntime

import "errors"

// Sentinel errors for dispatch registry.
var (
	// ErrEmptyServiceName is returned when registration is attempted with an empty serviceName.
	ErrEmptyServiceName = errors.New("rpcruntime: serviceName cannot be empty")

	// ErrNilHandler is returned when registration is attempted with a nil handler.
	ErrNilHandler = errors.New("rpcruntime: handler cannot be nil")
)
