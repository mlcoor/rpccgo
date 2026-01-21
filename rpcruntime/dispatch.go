package rpcruntime

import "sync"

// Protocol identifies the RPC protocol for handler registration.
type Protocol string

const (
	// ProtocolGrpc identifies gRPC handlers.
	ProtocolGrpc Protocol = "grpc"
	// ProtocolConnectRPC identifies connectrpc handlers.
	ProtocolConnectRPC Protocol = "connectrpc"
)

// handlerKey uniquely identifies a handler slot.
type handlerKey struct {
	protocol    Protocol
	serviceName string
}

var (
	handlerMu       sync.RWMutex
	handlerRegistry = make(map[handlerKey]any)
)

// RegisterGrpcHandler registers a gRPC handler for the given serviceName.
//
// If a handler is already registered for (grpc, serviceName), it is replaced.
// Returns replaced=true if an existing handler was overwritten.
// Returns an error if serviceName is empty or handler is nil.
func RegisterGrpcHandler(serviceName string, handler any) (replaced bool, err error) {
	return registerHandler(ProtocolGrpc, serviceName, handler)
}

// RegisterConnectHandler registers a connectrpc handler for the given serviceName.
//
// If a handler is already registered for (connectrpc, serviceName), it is replaced.
// Returns replaced=true if an existing handler was overwritten.
// Returns an error if serviceName is empty or handler is nil.
func RegisterConnectHandler(serviceName string, handler any) (replaced bool, err error) {
	return registerHandler(ProtocolConnectRPC, serviceName, handler)
}

// registerHandler is the internal implementation for handler registration.
func registerHandler(protocol Protocol, serviceName string, handler any) (replaced bool, err error) {
	if serviceName == "" {
		return false, ErrEmptyServiceName
	}
	if handler == nil {
		return false, ErrNilHandler
	}

	key := handlerKey{protocol: protocol, serviceName: serviceName}

	handlerMu.Lock()
	defer handlerMu.Unlock()

	_, existed := handlerRegistry[key]
	handlerRegistry[key] = handler

	return existed, nil
}

// LookupGrpcHandler looks up a gRPC handler for the given serviceName.
//
// Returns the handler and ok=true if found, otherwise nil and ok=false.
func LookupGrpcHandler(serviceName string) (handler any, ok bool) {
	return lookupHandler(ProtocolGrpc, serviceName)
}

// LookupConnectHandler looks up a connectrpc handler for the given serviceName.
//
// Returns the handler and ok=true if found, otherwise nil and ok=false.
func LookupConnectHandler(serviceName string) (handler any, ok bool) {
	return lookupHandler(ProtocolConnectRPC, serviceName)
}

// lookupHandler is the internal implementation for handler lookup.
func lookupHandler(protocol Protocol, serviceName string) (handler any, ok bool) {
	key := handlerKey{protocol: protocol, serviceName: serviceName}

	handlerMu.RLock()
	defer handlerMu.RUnlock()

	h, exists := handlerRegistry[key]
	return h, exists
}

// ListGrpcServices returns all registered gRPC service names.
//
// Useful for debugging and observability.
func ListGrpcServices() []string {
	return listServices(ProtocolGrpc)
}

// ListConnectServices returns all registered connectrpc service names.
//
// Useful for debugging and observability.
func ListConnectServices() []string {
	return listServices(ProtocolConnectRPC)
}

// listServices is the internal implementation for listing services.
func listServices(protocol Protocol) []string {
	handlerMu.RLock()
	defer handlerMu.RUnlock()

	var services []string
	for key := range handlerRegistry {
		if key.protocol == protocol {
			services = append(services, key.serviceName)
		}
	}

	return services
}

// clearHandlerRegistry clears the handler registry.
// This is intended for testing only.
func clearHandlerRegistry() {
	handlerMu.Lock()
	defer handlerMu.Unlock()

	handlerRegistry = make(map[handlerKey]any)
}
