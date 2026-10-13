// Package rpc provides the shared TCP and MessagePack transport for service RPCs.
package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

var (
	ErrInvalidAddress = errors.New("TCP listen address is required")
	ErrNilHandler     = errors.New("RPC handler is required")
	ErrNilContext     = errors.New("context must not be nil")
)

// Request is the common MessagePack RPC request envelope.
type Request struct {
	Method  string         `msgpack:"method"`
	Params  map[string]any `msgpack:"params"`
	TraceID string         `msgpack:"trace_id"`
	SpanID  string         `msgpack:"span_id"`
}

// Response is the common MessagePack RPC response envelope.
type Response struct {
	OK     bool           `msgpack:"ok"`
	Result map[string]any `msgpack:"result"`
	Error  *string        `msgpack:"error"`
}

// Handler handles one decoded RPC request.
type Handler func(context.Context, Request) Response

// ErrorResponse creates a standard unsuccessful RPC response.
func ErrorResponse(err error) Response {
	message := "unknown RPC error"
	if err != nil {
		message = err.Error()
	}
	return Response{OK: false, Error: &message}
}

// TCPServer accepts sequential MessagePack requests on each TCP connection.
// Requests from separate connections are handled concurrently.
type TCPServer struct {
	listener net.Listener
	handler  Handler

	closeOnce sync.Once
	closeErr  error
	connMu    sync.Mutex
	conns     map[net.Conn]struct{}
	handlers  sync.WaitGroup
}

// NewTCPServer listens on address and dispatches requests to handler.
func NewTCPServer(address string, handler Handler) (*TCPServer, error) {
	if address == "" {
		return nil, ErrInvalidAddress
	}
	if handler == nil {
		return nil, ErrNilHandler
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %q: %w", address, err)
	}
	return &TCPServer{
		listener: listener,
		handler:  handler,
		conns:    make(map[net.Conn]struct{}),
	}, nil
}

// Addr returns the address the server is listening on.
func (s *TCPServer) Addr() net.Addr {
	return s.listener.Addr()
}

// Serve accepts requests until ctx is canceled or the listener fails.
// Closing the server also closes active client connections.
func (s *TCPServer) Serve(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = s.Close()
		case <-watchDone:
		}
	}()
	defer close(watchDone)
	defer s.handlers.Wait()
	defer s.Close()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept RPC connection: %w", err)
		}

		s.connMu.Lock()
		s.conns[conn] = struct{}{}
		s.handlers.Add(1)
		s.connMu.Unlock()
		go s.handleConn(ctx, conn)
	}
}

// Close stops accepting connections and closes active clients.
func (s *TCPServer) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.listener.Close()
		if errors.Is(s.closeErr, net.ErrClosed) {
			s.closeErr = nil
		}
		s.connMu.Lock()
		for conn := range s.conns {
			_ = conn.Close()
		}
		s.connMu.Unlock()
	})
	return s.closeErr
}

func (s *TCPServer) handleConn(ctx context.Context, conn net.Conn) {
	defer s.handlers.Done()
	defer func() {
		s.connMu.Lock()
		delete(s.conns, conn)
		s.connMu.Unlock()
		_ = conn.Close()
	}()

	decoder := msgpack.NewDecoder(conn)
	encoder := msgpack.NewEncoder(conn)
	for {
		var request Request
		if err := decoder.Decode(&request); err != nil {
			return
		}
		if err := encoder.Encode(s.handler(ctx, request)); err != nil {
			return
		}
	}
}
