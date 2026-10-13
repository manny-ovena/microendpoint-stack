package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeWriter struct {
	messages    []kafka.Message
	writeErr    error
	closeErr    error
	closeCalls  int
	hasDeadline bool
}

func (w *fakeWriter) WriteMessages(ctx context.Context, messages ...kafka.Message) error {
	_, w.hasDeadline = ctx.Deadline()
	w.messages = append(w.messages, messages...)
	return w.writeErr
}

func (w *fakeWriter) Close() error {
	w.closeCalls++
	return w.closeErr
}

func TestNewRequiresBrokerAddresses(t *testing.T) {
	if _, err := New(Config{}); !errors.Is(err, ErrNoBrokers) {
		t.Fatalf("New() error = %v, want ErrNoBrokers", err)
	}
	if _, err := New(Config{Brokers: []string{" ", ""}}); !errors.Is(err, ErrNoBrokers) {
		t.Fatalf("New() with empty addresses error = %v, want ErrNoBrokers", err)
	}

	service, err := New(Config{Brokers: []string{"localhost:9092"}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv(EnvKafkaBrokers, " kafka-1:9092, kafka-2:9092 ,, ")

	config, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}
	if len(config.Brokers) != 2 || config.Brokers[0] != "kafka-1:9092" || config.Brokers[1] != "kafka-2:9092" {
		t.Errorf("ConfigFromEnv() brokers = %#v, want both trimmed addresses", config.Brokers)
	}

	t.Setenv(EnvKafkaBrokers, " , ")
	if _, err := ConfigFromEnv(); !errors.Is(err, ErrNoBrokers) {
		t.Errorf("ConfigFromEnv() error = %v, want ErrNoBrokers", err)
	}
}

func TestPublishWritesRecordAndWaitsForAcknowledgement(t *testing.T) {
	writer := &fakeWriter{}
	service := newWithWriter(writer)
	request := PublishRequest{
		Topic:   "events",
		Key:     []byte("key"),
		Payload: []byte("payload"),
		Headers: []Header{{Key: "trace", Value: []byte("trace-id")}},
		Options: DeliveryOptions{Timeout: time.Second},
	}

	response, err := service.Publish(context.Background(), request)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if response.Topic != request.Topic || !response.Acknowledged {
		t.Errorf("Publish() response = %#v, want acknowledged topic %q", response, request.Topic)
	}
	if !writer.hasDeadline {
		t.Error("Publish() did not apply the requested timeout")
	}
	if len(writer.messages) != 1 {
		t.Fatalf("writer received %d messages, want 1", len(writer.messages))
	}
	message := writer.messages[0]
	if message.Topic != request.Topic || string(message.Key) != "key" || string(message.Value) != "payload" {
		t.Errorf("writer message = %#v, want topic/key/payload from request", message)
	}
	if len(message.Headers) != 1 || message.Headers[0].Key != "trace" || string(message.Headers[0].Value) != "trace-id" {
		t.Errorf("writer headers = %#v, want trace header", message.Headers)
	}
}

func TestPublishReturnsWriterError(t *testing.T) {
	wantErr := errors.New("broker unavailable")
	service := newWithWriter(&fakeWriter{writeErr: wantErr})
	_, err := service.Publish(context.Background(), PublishRequest{Topic: "events"})
	if !errors.Is(err, wantErr) {
		t.Errorf("Publish() error = %v, want wrapped broker error", err)
	}
}

func TestPublishHonorsCanceledContext(t *testing.T) {
	writer := &fakeWriter{}
	service := newWithWriter(writer)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.Publish(ctx, PublishRequest{Topic: "events"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Publish() error = %v, want context.Canceled", err)
	}
	if len(writer.messages) != 0 {
		t.Errorf("canceled publish wrote %d messages, want 0", len(writer.messages))
	}
}

func TestPublishValidatesContextTopicAndTimeout(t *testing.T) {
	writer := &fakeWriter{}
	service := newWithWriter(writer)

	if _, err := service.Publish(nil, PublishRequest{Topic: "events"}); !errors.Is(err, ErrNilContext) {
		t.Errorf("Publish(nil context) error = %v, want ErrNilContext", err)
	}
	if _, err := service.Publish(context.Background(), PublishRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("Publish(empty topic) error = %v, want ErrInvalidRequest", err)
	}
	if _, err := service.Publish(context.Background(), PublishRequest{
		Topic: "events", Options: DeliveryOptions{Timeout: -1},
	}); !errors.Is(err, ErrInvalidTimeout) {
		t.Errorf("Publish(negative timeout) error = %v, want ErrInvalidTimeout", err)
	}
	if len(writer.messages) != 0 {
		t.Errorf("invalid requests wrote %d messages, want 0", len(writer.messages))
	}
}

func TestCloseClosesWriterOnceAndReturnsError(t *testing.T) {
	wantErr := errors.New("close failed")
	writer := &fakeWriter{closeErr: wantErr}
	service := newWithWriter(writer)

	for range 2 {
		if err := service.Close(); !errors.Is(err, wantErr) {
			t.Errorf("Close() error = %v, want close error", err)
		}
	}
	if writer.closeCalls != 1 {
		t.Errorf("writer Close() called %d times, want 1", writer.closeCalls)
	}
}
