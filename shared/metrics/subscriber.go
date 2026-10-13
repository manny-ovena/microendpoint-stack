package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// SubscriberMetrics groups metrics defined for Kafka subscribers in
// spec/observability.md.
type SubscriberMetrics struct {
	consumerLag   *prometheus.GaugeVec
	eventLatency  *prometheus.HistogramVec
	errors        *prometheus.CounterVec
	backlogEvents *prometheus.CounterVec
}

var subscriberRegistryMu sync.Mutex

// NewSubscriberMetrics creates the subscriber metric family for a service.
func NewSubscriberMetrics(registry *Registry, config Config) (*SubscriberMetrics, error) {
	if registry == nil {
		return nil, fmt.Errorf("subscriber metrics registry is required")
	}

	serviceName := strings.TrimSpace(config.ServiceName)
	if len(serviceName) <= len("subscriber-") || !strings.HasPrefix(serviceName, "subscriber-") {
		return nil, fmt.Errorf("invalid subscriber service name %q: want subscriber-*", config.ServiceName)
	}

	metrics := &SubscriberMetrics{
		consumerLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name:        "kafka_consumer_lag",
			Help:        "Number of Kafka messages the subscriber is behind.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"kafka_topic", "kafka_partition"}),
		eventLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "event_processing_latency_ms",
			Help:        "Time spent processing a Kafka event, in milliseconds.",
			Buckets:     []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"event_type", "kafka_topic"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "subscriber_errors_total",
			Help:        "Total number of subscriber errors.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"error_type", "dlq_reason"}),
		backlogEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "subscriber_backlog_events_total",
			Help:        "Total number of backlog events observed for the subscriber.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"kafka_topic"}),
	}

	collectors := []prometheus.Collector{
		metrics.consumerLag,
		metrics.eventLatency,
		metrics.errors,
		metrics.backlogEvents,
	}

	subscriberRegistryMu.Lock()
	defer subscriberRegistryMu.Unlock()

	if registry.registry == nil {
		registry.registry = prometheus.NewRegistry()
	}
	registered := make([]prometheus.Collector, 0, len(collectors))
	for _, collector := range collectors {
		if err := registry.registry.Register(collector); err != nil {
			for _, registeredCollector := range registered {
				registry.registry.Unregister(registeredCollector)
			}
			return nil, fmt.Errorf("register subscriber metric: %w", err)
		}
		registered = append(registered, collector)
	}

	return metrics, nil
}

// ObserveEventProcessing records processing latency for a bounded event type
// and topic.
func (m *SubscriberMetrics) ObserveEventProcessing(eventType, topic string, duration time.Duration) error {
	if m == nil || m.eventLatency == nil {
		return fmt.Errorf("subscriber event metrics are not initialized")
	}
	if !isSubscriberTopic(eventType) {
		return fmt.Errorf("invalid event type %q", eventType)
	}
	if !isSubscriberTopic(topic) {
		return fmt.Errorf("invalid Kafka topic %q", topic)
	}
	if duration < 0 {
		return fmt.Errorf("event processing duration must not be negative")
	}
	m.eventLatency.WithLabelValues(eventType, topic).Observe(float64(duration) / float64(time.Millisecond))
	return nil
}

// IncrementError records a subscriber error and its dead-letter reason.
func (m *SubscriberMetrics) IncrementError(errorType, deadLetterReason string) error {
	if m == nil || m.errors == nil {
		return fmt.Errorf("subscriber error metrics are not initialized")
	}
	switch errorType {
	case "json_decode", "rpc_error", "sql_error", "redis_error", "ipc_error":
	default:
		return fmt.Errorf("invalid subscriber error type %q", errorType)
	}
	switch deadLetterReason {
	case "null", "parse_error", "rpc_failure", "validation_error":
	default:
		return fmt.Errorf("invalid dead-letter reason %q", deadLetterReason)
	}
	m.errors.WithLabelValues(errorType, deadLetterReason).Inc()
	return nil
}

// SetConsumerLag records the lag for a topic partition.
func (m *SubscriberMetrics) SetConsumerLag(topic string, partition int, lag int64) error {
	if m == nil || m.consumerLag == nil {
		return fmt.Errorf("subscriber lag metrics are not initialized")
	}
	if !isSubscriberTopic(topic) {
		return fmt.Errorf("invalid Kafka topic %q", topic)
	}
	if partition < 0 {
		return fmt.Errorf("Kafka partition must not be negative")
	}
	if lag < 0 {
		return fmt.Errorf("Kafka consumer lag must not be negative")
	}
	m.consumerLag.WithLabelValues(topic, strconv.Itoa(partition)).Set(float64(lag))
	return nil
}

// AddBacklogEvents records newly observed backlog events for a topic.
func (m *SubscriberMetrics) AddBacklogEvents(topic string, count uint64) error {
	if m == nil || m.backlogEvents == nil {
		return fmt.Errorf("subscriber backlog metrics are not initialized")
	}
	if !isSubscriberTopic(topic) {
		return fmt.Errorf("invalid Kafka topic %q", topic)
	}
	m.backlogEvents.WithLabelValues(topic).Add(float64(count))
	return nil
}

func isSubscriberTopic(topic string) bool {
	switch topic {
	case "user.created", "order.placed", "inventory.changed":
		return true
	default:
		return false
	}
}
