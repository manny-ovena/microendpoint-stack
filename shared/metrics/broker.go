package metrics

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// BrokerMetrics groups metrics defined for SQL, Redis, and IPC brokers in
// spec/observability.md.
type BrokerMetrics struct {
	sqlQueryLatency     *prometheus.HistogramVec
	sqlPoolSaturation   *prometheus.GaugeVec
	redisCommandLatency *prometheus.HistogramVec
	redisPoolSaturation *prometheus.GaugeVec
	ipcOperationLatency *prometheus.HistogramVec
	ipcBytesWritten     *prometheus.CounterVec
	ipcBytesRead        *prometheus.CounterVec
}

var brokerRegistryMu sync.Mutex

// NewBrokerMetrics creates the broker metric family for a service.
func NewBrokerMetrics(registry *Registry, config Config) (*BrokerMetrics, error) {
	if registry == nil {
		return nil, fmt.Errorf("broker metrics registry is required")
	}

	serviceName := strings.TrimSpace(config.ServiceName)
	metrics := &BrokerMetrics{}
	var collectors []prometheus.Collector

	switch serviceName {
	case "broker-sql":
		metrics.sqlQueryLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "sql_query_latency_ms",
			Help:        "Latency of SQL query execution in milliseconds.",
			Buckets:     []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"sql_query_name", "error_type"})
		metrics.sqlPoolSaturation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name:        "sql_pool_saturation_pct",
			Help:        "SQL connection pool saturation percentage.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, nil)
		collectors = []prometheus.Collector{metrics.sqlQueryLatency, metrics.sqlPoolSaturation}
	case "broker-redis":
		metrics.redisCommandLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "redis_command_latency_ms",
			Help:        "Latency of Redis command execution in milliseconds.",
			Buckets:     []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"redis_command", "error_type"})
		metrics.redisPoolSaturation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name:        "redis_pool_saturation_pct",
			Help:        "Redis connection pool saturation percentage.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, nil)
		collectors = []prometheus.Collector{metrics.redisCommandLatency, metrics.redisPoolSaturation}
	case "broker-ipc":
		metrics.ipcOperationLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "ipc_operation_latency_ms",
			Help:        "Latency of IPC operations in milliseconds.",
			Buckets:     []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"ipc_operation", "error_type"})
		metrics.ipcBytesWritten = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "ipc_bytes_written_total",
			Help:        "Total bytes written through IPC.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, nil)
		metrics.ipcBytesRead = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "ipc_bytes_read_total",
			Help:        "Total bytes read through IPC.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, nil)
		collectors = []prometheus.Collector{
			metrics.ipcOperationLatency,
			metrics.ipcBytesWritten,
			metrics.ipcBytesRead,
		}
	default:
		return nil, fmt.Errorf("invalid broker service name %q: want broker-sql, broker-redis, or broker-ipc", config.ServiceName)
	}

	brokerRegistryMu.Lock()
	defer brokerRegistryMu.Unlock()

	if registry.registry == nil {
		registry.registry = prometheus.NewRegistry()
	}
	registered := make([]prometheus.Collector, 0, len(collectors))
	for _, collector := range collectors {
		if err := registry.registry.Register(collector); err != nil {
			for _, registeredCollector := range registered {
				registry.registry.Unregister(registeredCollector)
			}
			return nil, fmt.Errorf("register broker metric: %w", err)
		}
		registered = append(registered, collector)
	}

	return metrics, nil
}

// ObserveSQLQuery records the duration of a named SQL query.
func (m *BrokerMetrics) ObserveSQLQuery(queryName, errorType string, duration time.Duration) error {
	if m == nil || m.sqlQueryLatency == nil {
		return fmt.Errorf("SQL query metrics are not initialized")
	}
	switch queryName {
	case "insert_user", "insert_order", "update_inventory", "get_user", "get_order", "get_inventory":
	default:
		return fmt.Errorf("invalid SQL query name %q", queryName)
	}
	if errorType != "null" && errorType != "sql_error" {
		return fmt.Errorf("invalid SQL error type %q", errorType)
	}
	if duration < 0 {
		return fmt.Errorf("SQL query duration must not be negative")
	}
	m.sqlQueryLatency.WithLabelValues(queryName, errorType).
		Observe(float64(duration) / float64(time.Millisecond))
	return nil
}

// ObserveRedisCommand records the duration of a bounded Redis command.
func (m *BrokerMetrics) ObserveRedisCommand(command, errorType string, duration time.Duration) error {
	if m == nil || m.redisCommandLatency == nil {
		return fmt.Errorf("Redis command metrics are not initialized")
	}
	switch command {
	case "GET", "SET", "INCR", "DECR", "HGET", "HSET":
	default:
		return fmt.Errorf("invalid Redis command %q", command)
	}
	if errorType != "null" && errorType != "redis_error" {
		return fmt.Errorf("invalid Redis error type %q", errorType)
	}
	if duration < 0 {
		return fmt.Errorf("Redis command duration must not be negative")
	}
	m.redisCommandLatency.WithLabelValues(command, errorType).
		Observe(float64(duration) / float64(time.Millisecond))
	return nil
}

// ObserveIPCOperation records the duration of a bounded IPC operation.
func (m *BrokerMetrics) ObserveIPCOperation(operation, errorType string, duration time.Duration) error {
	if m == nil || m.ipcOperationLatency == nil {
		return fmt.Errorf("IPC operation metrics are not initialized")
	}
	switch operation {
	case "read", "write", "delete", "list":
	default:
		return fmt.Errorf("invalid IPC operation %q", operation)
	}
	if errorType != "null" && errorType != "ipc_error" {
		return fmt.Errorf("invalid IPC error type %q", errorType)
	}
	if duration < 0 {
		return fmt.Errorf("IPC operation duration must not be negative")
	}
	m.ipcOperationLatency.WithLabelValues(operation, errorType).
		Observe(float64(duration) / float64(time.Millisecond))
	return nil
}

// SetSQLPoolSaturation records SQL connection pool saturation as a percentage.
func (m *BrokerMetrics) SetSQLPoolSaturation(saturation float64) error {
	if m == nil || m.sqlPoolSaturation == nil {
		return fmt.Errorf("SQL pool saturation metric is not initialized")
	}
	if !isValidSaturation(saturation) {
		return fmt.Errorf("SQL pool saturation must be between 0 and 100")
	}
	m.sqlPoolSaturation.WithLabelValues().Set(saturation)
	return nil
}

// SetRedisPoolSaturation records Redis pool saturation as a percentage.
func (m *BrokerMetrics) SetRedisPoolSaturation(saturation float64) error {
	if m == nil || m.redisPoolSaturation == nil {
		return fmt.Errorf("Redis pool saturation metric is not initialized")
	}
	if !isValidSaturation(saturation) {
		return fmt.Errorf("Redis pool saturation must be between 0 and 100")
	}
	m.redisPoolSaturation.WithLabelValues().Set(saturation)
	return nil
}

// AddIPCBytesWritten records bytes written through IPC.
func (m *BrokerMetrics) AddIPCBytesWritten(bytes uint64) error {
	if m == nil || m.ipcBytesWritten == nil {
		return fmt.Errorf("IPC bytes written metric is not initialized")
	}
	m.ipcBytesWritten.WithLabelValues().Add(float64(bytes))
	return nil
}

// AddIPCBytesRead records bytes read through IPC.
func (m *BrokerMetrics) AddIPCBytesRead(bytes uint64) error {
	if m == nil || m.ipcBytesRead == nil {
		return fmt.Errorf("IPC bytes read metric is not initialized")
	}
	m.ipcBytesRead.WithLabelValues().Add(float64(bytes))
	return nil
}

func isValidSaturation(saturation float64) bool {
	return !math.IsNaN(saturation) && !math.IsInf(saturation, 0) && saturation >= 0 && saturation <= 100
}
