package rpc

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

func TestTCPServerHandlesMultipleRequestsOnOneConnection(t *testing.T) {
	server, err := NewTCPServer("127.0.0.1:0", func(_ context.Context, request Request) Response {
		if request.Method != "echo" {
			return ErrorResponse(errors.New("unknown method"))
		}
		result := request.Params
		result["trace_id"] = request.TraceID
		result["span_id"] = request.SpanID
		return Response{OK: true, Result: result}
	})
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(ctx) }()

	conn, err := net.DialTimeout("tcp", server.Addr().String(), time.Second)
	if err != nil {
		cancel()
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()
	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	for _, want := range []string{"first", "second"} {
		request := Request{
			Method:  "echo",
			Params:  map[string]any{"value": want},
			TraceID: "trace-1",
			SpanID:  "span-1",
		}
		if err := encoder.Encode(request); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		var response Response
		if err := decoder.Decode(&response); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if !response.OK || response.Result["value"] != want {
			t.Errorf("response = %#v, want successful value %q", response, want)
		}
		if response.Result["trace_id"] != "trace-1" || response.Result["span_id"] != "span-1" {
			t.Errorf("request trace metadata = %#v, want trace-1/span-1", response.Result)
		}
	}

	cancel()
	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not stop after context cancellation")
	}
}

func TestTCPServerClosesIdleConnectionsOnCancellation(t *testing.T) {
	server, err := NewTCPServer("127.0.0.1:0", func(context.Context, Request) Response {
		return Response{OK: true}
	})
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(ctx) }()

	conn, err := net.DialTimeout("tcp", server.Addr().String(), time.Second)
	if err != nil {
		cancel()
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	cancel()
	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not close idle client on cancellation")
	}
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("client connection remained open after context cancellation")
	}
}

func TestNewTCPServerValidatesInputs(t *testing.T) {
	handler := func(context.Context, Request) Response { return Response{OK: true} }
	if _, err := NewTCPServer("", handler); !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("NewTCPServer(empty address) error = %v, want ErrInvalidAddress", err)
	}
	if _, err := NewTCPServer("127.0.0.1:0", nil); !errors.Is(err, ErrNilHandler) {
		t.Errorf("NewTCPServer(nil handler) error = %v, want ErrNilHandler", err)
	}

	server, err := NewTCPServer("127.0.0.1:0", handler)
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}
	defer server.Close()
	if err := server.Serve(nil); !errors.Is(err, ErrNilContext) {
		t.Errorf("Serve(nil) error = %v, want ErrNilContext", err)
	}
}
