package rpcruntime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// StreamHandle represents a unique identifier for an in-flight stream session.
type StreamHandle uint64

// StreamSession is the exported interface for stream session access by generated adaptors.
type StreamSession interface {
	Context() context.Context
	Cancel()
	Protocol() Protocol
	SendCh() chan any
	RespCh() chan streamResult
	SetHandlerState(state any)
	HandlerState() any
	SetCallbacks(onRead func(any) bool, onDone func(error))
	OnRead() func(any) bool
	OnDone() func(error)
}

// streamSession holds the state for a streaming call.
type streamSession struct {
	ctx      context.Context
	cancel   context.CancelFunc
	protocol Protocol
	finished bool

	// For client-streaming and bidi: channel to send requests.
	sendCh       chan any
	sendChClosed bool

	// For server-streaming and bidi: callbacks.
	onRead func(any) bool
	onDone func(error)

	// For client-streaming: channel to receive final response.
	respCh chan streamResult

	// Handler-specific state (grpc stream, connect stream, etc.)
	handlerState any
}

type streamResult struct {
	resp any
	err  error
}

var (
	streamMu       sync.RWMutex
	streamRegistry = make(map[StreamHandle]*streamSession)
	nextStreamID   atomic.Uint64
)

// allocateStreamHandle creates a new stream session and returns its handle.
func AllocateStreamHandle(ctx context.Context, protocol Protocol) (StreamHandle, context.Context, context.CancelFunc) {
	id := StreamHandle(nextStreamID.Add(1))
	childCtx, cancel := context.WithCancel(ctx)

	session := &streamSession{
		ctx:      childCtx,
		cancel:   cancel,
		protocol: protocol,
		sendCh:   make(chan any, 16), // Buffered to avoid blocking.
		respCh:   make(chan streamResult, 1),
	}

	streamMu.Lock()
	streamRegistry[id] = session
	streamMu.Unlock()

	return id, childCtx, cancel
}

// GetStreamSession retrieves a stream session by handle.
// Returns nil if not found or already finished.
func GetStreamSession(handle StreamHandle) StreamSession {
	return getStreamSessionInternal(handle)
}

// getStreamSessionInternal returns the concrete session for internal use.
func getStreamSessionInternal(handle StreamHandle) *streamSession {
	streamMu.RLock()
	defer streamMu.RUnlock()

	session, ok := streamRegistry[handle]
	if !ok || session.finished {
		return nil
	}
	return session
}

// FinishStreamHandle marks a stream as finished and removes it from registry.
func FinishStreamHandle(handle StreamHandle) {
	streamMu.Lock()
	defer streamMu.Unlock()

	if session, ok := streamRegistry[handle]; ok {
		session.finished = true
		session.cancel()
		if !session.sendChClosed {
			session.sendChClosed = true
			close(session.sendCh)
		}
		delete(streamRegistry, handle)
	}
}

// StreamSession accessors.
func (s *streamSession) Context() context.Context  { return s.ctx }
func (s *streamSession) Cancel()                   { s.cancel() }
func (s *streamSession) Protocol() Protocol        { return s.protocol }
func (s *streamSession) SendCh() chan any          { return s.sendCh }
func (s *streamSession) RespCh() chan streamResult { return s.respCh }
func (s *streamSession) SetHandlerState(state any) { s.handlerState = state }
func (s *streamSession) HandlerState() any         { return s.handlerState }
func (s *streamSession) SetCallbacks(onRead func(any) bool, onDone func(error)) {
	s.onRead = onRead
	s.onDone = onDone
}
func (s *streamSession) OnRead() func(any) bool { return s.onRead }
func (s *streamSession) OnDone() func(error)    { return s.onDone }

// CloseSendCh safely closes the send channel (called from bidi CloseSend).
func CloseSendCh(handle StreamHandle) error {
	streamMu.Lock()
	defer streamMu.Unlock()

	session, ok := streamRegistry[handle]
	if !ok || session.finished {
		return ErrInvalidStreamHandle
	}

	if !session.sendChClosed {
		session.sendChClosed = true
		close(session.sendCh)
	}
	return nil
}

// SendToStream sends a message to the stream's send channel.
// Returns ErrInvalidStreamHandle if the session is invalid or finished.
func SendToStream(handle StreamHandle, msg any) error {
	session := getStreamSessionInternal(handle)
	if session == nil {
		return ErrInvalidStreamHandle
	}

	select {
	case session.sendCh <- msg:
		return nil
	case <-session.ctx.Done():
		return session.ctx.Err()
	}
}

// FinishClientStream signals the end of client-side sending and waits for response.
// Returns the response and any error.
func FinishClientStream(handle StreamHandle) (any, error) {
	session := getStreamSessionInternal(handle)
	if session == nil {
		return nil, ErrInvalidStreamHandle
	}

	// Close send channel to signal end of sending (with lock to prevent double close).
	streamMu.Lock()
	if !session.sendChClosed {
		session.sendChClosed = true
		close(session.sendCh)
	}
	streamMu.Unlock()

	// Wait for response.
	select {
	case result := <-session.respCh:
		FinishStreamHandle(handle)
		return result.resp, result.err
	case <-session.ctx.Done():
		FinishStreamHandle(handle)
		return nil, session.ctx.Err()
	}
}

// CompleteClientStream is called by handler goroutine to send the final response.
func CompleteClientStream(handle StreamHandle, resp any, err error) {
	session := getStreamSessionInternal(handle)
	if session == nil {
		return
	}

	select {
	case session.respCh <- streamResult{resp: resp, err: err}:
	default:
		// Response channel full or closed, ignore.
	}
}

// clearStreamRegistry clears all stream sessions.
// This is intended for testing only.
func clearStreamRegistry() {
	streamMu.Lock()
	defer streamMu.Unlock()

	for id, session := range streamRegistry {
		session.cancel()
		delete(streamRegistry, id)
	}
}

// RecoverPanic converts a recovered panic value to an error.
// Use this in defer to safely handle panics in handler goroutines.
func RecoverPanic(r any) error {
	if r == nil {
		return nil
	}
	if err, ok := r.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", r)
}
