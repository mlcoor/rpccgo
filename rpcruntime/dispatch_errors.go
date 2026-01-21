package rpcruntime

import "errors"

// Sentinel errors for dispatch registry.
var (
	// ErrEmptyServiceName is returned when registration is attempted with an empty serviceName.
	ErrEmptyServiceName = errors.New("rpcruntime: serviceName cannot be empty")

	// ErrNilHandler is returned when registration is attempted with a nil handler.
	ErrNilHandler = errors.New("rpcruntime: handler cannot be nil")
)

// Sentinel errors for adaptor dispatch.
var (
	// ErrMissingProtocol is returned when protocol is not set in context.
	ErrMissingProtocol = errors.New("rpcruntime: protocol not set in context")

	// ErrUnknownProtocol is returned when protocol is not grpc or connectrpc.
	ErrUnknownProtocol = errors.New("rpcruntime: unknown protocol")

	// ErrServiceNotRegistered is returned when no handler is registered for the service.
	ErrServiceNotRegistered = errors.New("rpcruntime: service not registered")

	// ErrHandlerTypeMismatch is returned when handler cannot be asserted to expected type.
	ErrHandlerTypeMismatch = errors.New("rpcruntime: handler type mismatch")

	// ErrInvalidStreamHandle is returned when stream handle is invalid or already finished.
	ErrInvalidStreamHandle = errors.New("rpcruntime: invalid or finished stream handle")
)
