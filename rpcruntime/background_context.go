package rpcruntime

import (
	"context"
)

// BackgroundContext returns a context.Context intended to be used by generated CGO entrypoints.
//
// It always carries a protocol selection (see WithProtocol / ProtocolFromContext).
//
// Selection rules:
//   - If a default protocol has been set via SetDefaultProtocol, uses it.
//   - Otherwise returns context.Background() without a protocol value.
func BackgroundContext() context.Context {
	ctx := context.Background()
	if p, ok := DefaultProtocol(); ok {
		return WithProtocol(ctx, p)
	}
	return ctx
}
