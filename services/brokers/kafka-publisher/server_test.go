package service

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

func TestTCPServerPublishesMessagePackRequest(t *testing.T) {
	writer := &fakeWriter{}
	publisher := newWithWriter(writer)
	server, err := NewTCPServer("127.0.0.1:0", publisher)
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(ctx)
	}()

	conn, err := net.DialTimeout("tcp", server.Addr().String(), time.Second)
	if err != nil {
		cancel()
		t.Fatalf("Dial() error = %v", err)
	}
	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	request := rpcRequest{
		Method: "Publish",
		Params: map[string]any{
			"topic":      "events",
			"key":        []byte("key-1"),
			"payload":    []byte(`{"id":1}`),
			"timeout_ms": int64(1000),
		},
		TraceID: "trace-1",
		SpanID:  "span-1",
	}
	if err := encoder.Encode(request); err != nil {
		_ = conn.Close()
		cancel()
		t.Fatalf("Encode() error = %v", err)
	}
	var response rpcResponse
	if err := decoder.Decode(&response); err != nil {
		_ = conn.Close()
		cancel()
		t.Fatalf("Decode() error = %v", err)
	}
	if !response.OK || response.Error != nil {
		t.Fatalf("RPC response = %#v, want success", response)
	}
	if response.Result["topic"] != "events" || response.Result["acknowledged"] != true {
		t.Errorf("RPC result = %#v, want acknowledged topic", response.Result)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("published %d messages, want 1", len(writer.messages))
	}
	if writer.messages[0].Topic != "events" || string(writer.messages[0].Key) != "key-1" || string(writer.messages[0].Value) != `{"id":1}` {
		t.Errorf("published message = %#v, want request contents", writer.messages[0])
	}

	_ = conn.Close()
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

func TestTCPServerReturnsRPCErrorForUnknownMethod(t *testing.T) {
	server, err := NewTCPServer("127.0.0.1:0", newWithWriter(&fakeWriter{}))
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}
	defer server.Close()

	response := server.dispatch(context.Background(), rpcRequest{Method: "GetUser"})
	if response.OK || response.Error == nil {
		t.Fatalf("dispatch() = %#v, want RPC error", response)
	}
}
