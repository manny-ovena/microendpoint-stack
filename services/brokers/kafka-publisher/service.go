// Package service contains the Kafka publishing broker.
package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	ErrNoBrokers      = errors.New("Kafka broker list is required")
	ErrInvalidRequest = errors.New("invalid Kafka publish request")
	ErrNilContext     = errors.New("context must not be nil")
	ErrInvalidTimeout = errors.New("publish timeout must not be negative")
)

const EnvKafkaBrokers = "KAFKA_BROKERS"

// Config contains the Kafka bootstrap broker addresses.
type Config struct {
	Brokers []string
}

// ConfigFromEnv loads comma-separated Kafka bootstrap addresses from
// KAFKA_BROKERS.
func ConfigFromEnv() (Config, error) {
	raw := strings.TrimSpace(os.Getenv(EnvKafkaBrokers))
	if raw == "" {
		return Config{}, ErrNoBrokers
	}

	var brokers []string
	for _, broker := range strings.Split(raw, ",") {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	if len(brokers) == 0 {
		return Config{}, ErrNoBrokers
	}
	return Config{Brokers: brokers}, nil
}

// PublishRequest describes a Kafka record to publish.
type PublishRequest struct {
	Topic   string
	Key     []byte
	Payload []byte
	Headers []Header
	Options DeliveryOptions
}

// PublishParams is the MessagePack RPC representation of a publish request.
type PublishParams struct {
	Topic     string   `msgpack:"topic"`
	Key       []byte   `msgpack:"key"`
	Payload   []byte   `msgpack:"payload"`
	Headers   []Header `msgpack:"headers"`
	TimeoutMS int64    `msgpack:"timeout_ms"`
}

// Header is a Kafka record header. A slice allows repeated header keys.
type Header struct {
	Key   string `msgpack:"key"`
	Value []byte `msgpack:"value"`
}

// DeliveryOptions contains per-message delivery settings. A zero Timeout
// leaves the producer's configured timeout unchanged.
type DeliveryOptions struct {
	Timeout time.Duration
}

// PublishResponse reports Kafka's acknowledgement for a successfully
// delivered record. Delivery failures are returned as errors.
type PublishResponse struct {
	Topic        string
	Acknowledged bool
}

type messageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
	Close() error
}

// Service is the Kafka publishing broker. Its writer is created once and
// reused for every message published by this service instance.
type Service struct {
	writer    messageWriter
	closeOnce sync.Once
	closeErr  error
}

// New creates a Kafka publishing service with one reusable synchronous writer.
// Publish returns only after Kafka acknowledges the message from all
// in-sync replicas. The effective durability is bounded by the topic's
// replication factor.
func New(config Config) (*Service, error) {
	brokers := make([]string, 0, len(config.Brokers))
	for _, broker := range config.Brokers {
		if broker = strings.TrimSpace(broker); broker != "" {
			brokers = append(brokers, broker)
		}
	}
	if len(brokers) == 0 {
		return nil, ErrNoBrokers
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	return newWithWriter(writer), nil
}

// NewFromEnv creates a Kafka publishing service using KAFKA_BROKERS.
func NewFromEnv() (*Service, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return New(config)
}

func newWithWriter(writer messageWriter) *Service {
	return &Service{writer: writer}
}

// Publish writes a record to Kafka and waits for the configured broker
// acknowledgement before returning success.
func (s *Service) Publish(ctx context.Context, req PublishRequest) (PublishResponse, error) {
	if ctx == nil {
		return PublishResponse{}, ErrNilContext
	}
	if strings.TrimSpace(req.Topic) == "" {
		return PublishResponse{}, fmt.Errorf("%w: topic is required", ErrInvalidRequest)
	}
	if req.Options.Timeout < 0 {
		return PublishResponse{}, ErrInvalidTimeout
	}

	if req.Options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Options.Timeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return PublishResponse{}, fmt.Errorf("publish Kafka message to topic %q: %w", req.Topic, err)
	}

	headers := make([]kafka.Header, len(req.Headers))
	for i, header := range req.Headers {
		headers[i] = kafka.Header{Key: header.Key, Value: header.Value}
	}
	message := kafka.Message{
		Topic:   req.Topic,
		Key:     req.Key,
		Value:   req.Payload,
		Headers: headers,
	}
	if err := s.writer.WriteMessages(ctx, message); err != nil {
		return PublishResponse{}, fmt.Errorf("publish Kafka message to topic %q: %w", req.Topic, err)
	}

	return PublishResponse{Topic: req.Topic, Acknowledged: true}, nil
}

// Close flushes outstanding writes and releases the reusable Kafka writer.
// It is safe to call Close more than once.
func (s *Service) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.writer.Close()
	})
	return s.closeErr
}
