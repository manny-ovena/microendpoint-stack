package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// RPCMetrics groups metrics defined for endpoint and subscriber RPC calls in
// spec/observability.md.
type RPCMetrics struct {
	latency    *prometheus.HistogramVec
	queueDepth *prometheus.GaugeVec
}

var rpcRegistryMu sync.Mutex

// NewRPCMetrics creates the RPC metric family for a service.
func NewRPCMetrics(registry *Registry, config Config) (*RPCMetrics, error) {
	if registry == nil {
		return nil, fmt.Errorf("RPC metrics registry is required")
	}

	serviceName := strings.TrimSpace(config.ServiceName)
	if !isRPCServiceName(serviceName) {
		return nil, fmt.Errorf("invalid RPC service name %q: want endpoint-* or subscriber-*", config.ServiceName)
	}

	metrics := &RPCMetrics{
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "rpc_latency_ms",
			Help:        "Latency of RPC calls in milliseconds.",
			Buckets:     []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, []string{"rpc_method", "error_type"}),
		queueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name:        "rpc_queue_depth",
			Help:        "Current depth of the RPC queue.",
			ConstLabels: prometheus.Labels{"service_name": serviceName},
		}, nil),
	}
	collectors := []prometheus.Collector{metrics.latency, metrics.queueDepth}

	rpcRegistryMu.Lock()
	defer rpcRegistryMu.Unlock()

	if registry.registry == nil {
		registry.registry = prometheus.NewRegistry()
	}
	registered := make([]prometheus.Collector, 0, len(collectors))
	for _, collector := range collectors {
		if err := registry.registry.Register(collector); err != nil {
			for _, registeredCollector := range registered {
				registry.registry.Unregister(registeredCollector)
			}
			return nil, fmt.Errorf("register RPC metric: %w", err)
		}
		registered = append(registered, collector)
	}

	return metrics, nil
}

// ObserveRPCCall records the duration of a bounded RPC method.
func (m *RPCMetrics) ObserveRPCCall(method, errorType string, duration time.Duration) error {
	if m == nil || m.latency == nil {
		return fmt.Errorf("RPC latency metrics are not initialized")
	}
	switch method {
	case "InsertUser", "InsertOrder", "UpdateInventory", "GetUser", "GetOrder", "GetInventory":
	default:
		return fmt.Errorf("invalid RPC method %q", method)
	}
	switch errorType {
	case "null", "rpc_error", "sql_error", "redis_error", "ipc_error":
	default:
		return fmt.Errorf("invalid RPC error type %q", errorType)
	}
	if duration < 0 {
		return fmt.Errorf("RPC call duration must not be negative")
	}
	m.latency.WithLabelValues(method, errorType).
		Observe(float64(duration) / float64(time.Millisecond))
	return nil
}

// SetQueueDepth records the current RPC queue depth.
func (m *RPCMetrics) SetQueueDepth(depth int64) error {
	if m == nil || m.queueDepth == nil {
		return fmt.Errorf("RPC queue depth metric is not initialized")
	}
	if depth < 0 {
		return fmt.Errorf("RPC queue depth must not be negative")
	}
	m.queueDepth.WithLabelValues().Set(float64(depth))
	return nil
}

func isRPCServiceName(serviceName string) bool {
	return strings.HasPrefix(serviceName, "endpoint-") && len(serviceName) > len("endpoint-") ||
		strings.HasPrefix(serviceName, "subscriber-") && len(serviceName) > len("subscriber-")
}
