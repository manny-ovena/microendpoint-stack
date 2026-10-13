package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

var (
	ErrInvalidAddress = errors.New("TCP listen address is required")
	ErrUnknownMethod  = errors.New("unknown RPC method")
	ErrInvalidParams  = errors.New("invalid Publish params")
)

type rpcRequest struct {
	Method  string         `msgpack:"method"`
	Params  map[string]any `msgpack:"params"`
	TraceID string         `msgpack:"trace_id"`
	SpanID  string         `msgpack:"span_id"`
}

type rpcResponse struct {
	OK     bool           `msgpack:"ok"`
	Result map[string]any `msgpack:"result"`
	Error  *string        `msgpack:"error"`
}

// TCPServer accepts MessagePack RPC requests and dispatches Publish calls to
// the service. Each connection may carry multiple request/response objects.
type TCPServer struct {
	listener net.Listener
	service  *Service

	closeOnce sync.Once
	connMu    sync.Mutex
	conns     map[net.Conn]struct{}
	handlers  sync.WaitGroup
}

// NewTCPServer listens on address and serves the supplied publisher.
func NewTCPServer(address string, publisher *Service) (*TCPServer, error) {
	if address == "" {
		return nil, ErrInvalidAddress
	}
	if publisher == nil {
		return nil, errors.New("publisher service is required")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %q: %w", address, err)
	}
	return &TCPServer{
		listener: listener,
		service:  publisher,
		conns:    make(map[net.Conn]struct{}),
	}, nil
}

// Addr returns the address the server is listening on.
func (s *TCPServer) Addr() net.Addr {
	return s.listener.Addr()
}

// Serve accepts MessagePack requests until ctx is canceled or the listener
// fails. Closing the server unblocks active request handlers.
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
	defer s.Close()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				break
			}
			return fmt.Errorf("accept RPC connection: %w", err)
		}

		s.connMu.Lock()
		s.conns[conn] = struct{}{}
		s.handlers.Add(1)
		s.connMu.Unlock()
		go s.handleConn(ctx, conn)
	}

	s.handlers.Wait()
	return nil
}

// Close stops accepting connections and closes active clients.
func (s *TCPServer) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.listener.Close()
		s.connMu.Lock()
		for conn := range s.conns {
			_ = conn.Close()
		}
		s.connMu.Unlock()
	})
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
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
		var request rpcRequest
		if err := decoder.Decode(&request); err != nil {
			return
		}
		response := s.dispatch(ctx, request)
		if err := encoder.Encode(response); err != nil {
			return
		}
	}
}

func (s *TCPServer) dispatch(ctx context.Context, request rpcRequest) rpcResponse {
	if request.Method != "Publish" {
		return rpcError(fmt.Errorf("%w: %s", ErrUnknownMethod, request.Method))
	}
	params, err := decodePublishParams(request.Params)
	if err != nil {
		return rpcError(err)
	}
	if params.TimeoutMS < 0 || params.TimeoutMS > math.MaxInt64/int64(time.Millisecond) {
		return rpcError(ErrInvalidTimeout)
	}
	publishRequest := PublishRequest{
		Topic:   params.Topic,
		Key:     params.Key,
		Payload: params.Payload,
		Headers: params.Headers,
		Options: DeliveryOptions{Timeout: time.Duration(params.TimeoutMS) * time.Millisecond},
	}
	result, err := s.service.Publish(ctx, publishRequest)
	if err != nil {
		return rpcError(err)
	}
	return rpcResponse{
		OK: true,
		Result: map[string]any{
			"topic":        result.Topic,
			"acknowledged": result.Acknowledged,
		},
	}
}

func decodePublishParams(params map[string]any) (PublishParams, error) {
	if params == nil {
		return PublishParams{}, fmt.Errorf("%w: params are required", ErrInvalidParams)
	}
	encoded, err := msgpack.Marshal(params)
	if err != nil {
		return PublishParams{}, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	var decoded PublishParams
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		return PublishParams{}, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	return decoded, nil
}

func rpcError(err error) rpcResponse {
	message := err.Error()
	return rpcResponse{OK: false, Error: &message}
}
