package rpcruntime

import "sync/atomic"

type defaultProtocolState struct {
	set      bool
	protocol Protocol
}

var defaultProtocol atomic.Value

func init() {
	defaultProtocol.Store(defaultProtocolState{})
}

// SetDefaultProtocol sets the default protocol used by BackgroundContext.
//
// Passing Protocol(""), or calling ClearDefaultProtocol, leaves BackgroundContext without a protocol value.
func SetDefaultProtocol(protocol Protocol) error {
	switch protocol {
	case ProtocolGrpc, ProtocolConnectRPC:
		defaultProtocol.Store(defaultProtocolState{set: true, protocol: protocol})
		return nil
	case "":
		defaultProtocol.Store(defaultProtocolState{})
		return nil
	default:
		return ErrUnknownProtocol
	}
}

// ClearDefaultProtocol clears any default protocol selection.
func ClearDefaultProtocol() {
	defaultProtocol.Store(defaultProtocolState{})
}

// DefaultProtocol returns the current default protocol and whether it is set.
func DefaultProtocol() (Protocol, bool) {
	st := defaultProtocol.Load().(defaultProtocolState)
	return st.protocol, st.set
}
