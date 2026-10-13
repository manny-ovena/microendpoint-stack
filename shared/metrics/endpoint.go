package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// EndpointMetrics groups metrics defined for HTTP endpoints in
// spec/observability.md.
type EndpointMetrics struct {
	requestLatency *prometheus.HistogramVec
	requests       *prometheus.CounterVec
	errors         *prometheus.CounterVec
	cpuUsage       *prometheus.CounterVec
	memoryUsage    *prometheus.GaugeVec
}

var endpointRegistryMu sync.Mutex

// NewEndpointMetrics creates the endpoint metric family for a service.
func NewEndpointMetrics(registry *Registry, config Config) (*EndpointMetrics, error) {
	if registry == nil {
		return nil, fmt.Errorf("endpoint metrics registry is required")
	}

	serviceName := strings.TrimSpace(config.ServiceName)
	if len(serviceName) <= len("endpoint-") || !strings.HasPrefix(serviceName, "endpoint-") {
		return nil, fmt.Errorf("invalid endpoint service name %q: want endpoint-*", config.ServiceName)
	}

	metrics := &EndpointMetrics{
		requestLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_latency_ms",
			Help:    "Total HTTP request latency in milliseconds.",
			Buckets: []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
			ConstLabels: prometheus.Labels{
				"service_name": serviceName,
			},
		}, []string{"http_method", "http_path", "status_code", "rpc_target"}),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
			ConstLabels: prometheus.Labels{
				"service_name": serviceName,
			},
		}, []string{"http_method", "http_path"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors.",
			ConstLabels: prometheus.Labels{
				"service_name": serviceName,
			},
		}, []string{"error_type", "rpc_target"}),
		cpuUsage: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "endpoint_cpu_seconds_total",
			Help: "Total CPU usage in seconds.",
			ConstLabels: prometheus.Labels{
				"service_name": serviceName,
			},
		}, []string{"pod_name"}),
		memoryUsage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "endpoint_memory_bytes",
			Help: "Memory usage in bytes.",
			ConstLabels: prometheus.Labels{
				"service_name": serviceName,
			},
		}, []string{"pod_name"}),
	}

	collectors := []prometheus.Collector{
		metrics.requestLatency,
		metrics.requests,
		metrics.errors,
		metrics.cpuUsage,
		metrics.memoryUsage,
	}

	endpointRegistryMu.Lock()
	defer endpointRegistryMu.Unlock()

	if registry.registry == nil {
		registry.registry = prometheus.NewRegistry()
	}
	registered := make([]prometheus.Collector, 0, len(collectors))
	for _, collector := range collectors {
		if err := registry.registry.Register(collector); err != nil {
			for _, registeredCollector := range registered {
				registry.registry.Unregister(registeredCollector)
			}
			return nil, fmt.Errorf("failed to register collector: %w", err)
		}
		registered = append(registered, collector)
	}

	return metrics, nil
}

// ObserveHTTPRequest records a request using a normalized path and bounded
// method and status values.
func (m *EndpointMetrics) ObserveHTTPRequest(method, path string, statusCode int, rpcTarget string, duration time.Duration) error {
	if m == nil || m.requestLatency == nil {
		return fmt.Errorf("endpoint request latency metric is not initialized")
	}
	if duration < 0 {
		return fmt.Errorf("invalid request duration: %v", duration)
	}

	method = strings.ToUpper(method)
	if !isEndpointMethod(method) {
		return fmt.Errorf("invalid HTTP method %q", method)
	}
	if !isEndpointStatusCode(statusCode) {
		return fmt.Errorf("invalid HTTP status code %d", statusCode)
	}
	if !isEndpointRPCTarget(rpcTarget) {
		return fmt.Errorf("invalid RPC target %q", rpcTarget)
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("normalized HTTP path is required")
	}

	m.requests.WithLabelValues(method, path).Inc()
	m.requestLatency.WithLabelValues(method, path, fmt.Sprintf("%d", statusCode), rpcTarget).
		Observe(float64(duration) / float64(time.Millisecond))
	return nil
}

// IncrementHTTPError records an endpoint error and its downstream target.
func (m *EndpointMetrics) IncrementHTTPError(errorType, rpcTarget string) error {
	if m == nil || m.errors == nil {
		return fmt.Errorf("endpoint errors metric is not initialized")
	}
	if !isEndpointErrorType(errorType) {
		return fmt.Errorf("invalid HTTP error type %q", errorType)
	}
	if !isEndpointRPCTarget(rpcTarget) {
		return fmt.Errorf("invalid RPC target %q", rpcTarget)
	}
	m.errors.WithLabelValues(errorType, rpcTarget).Inc()
	return nil
}

// AddCPUTime records CPU time used by the endpoint process.
func (m *EndpointMetrics) AddCPUTime(podName string, duration time.Duration) error {
	if m == nil || m.cpuUsage == nil {
		return fmt.Errorf("endpoint CPU usage metric is not initialized")
	}
	if strings.TrimSpace(podName) == "" {
		return fmt.Errorf("pod name is required")
	}
	if duration < 0 {
		return fmt.Errorf("invalid CPU duration: %v", duration)
	}
	m.cpuUsage.WithLabelValues(podName).Add(float64(duration) / float64(time.Second))
	return nil
}

// SetMemoryUsage records the endpoint process memory usage in bytes.
func (m *EndpointMetrics) SetMemoryUsage(podName string, bytes uint64) error {
	if m == nil || m.memoryUsage == nil {
		return fmt.Errorf("endpoint memory usage metric is not initialized")
	}
	if strings.TrimSpace(podName) == "" {
		return fmt.Errorf("pod name is required")
	}
	m.memoryUsage.WithLabelValues(podName).Set(float64(bytes))
	return nil
}

func isEndpointMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

func isEndpointStatusCode(statusCode int) bool {
	switch statusCode {
	case 200, 400, 401, 403, 404, 409, 500:
		return true
	default:
		return false
	}
}

func isEndpointErrorType(errorType string) bool {
	switch errorType {
	case "json_decode", "rpc_error", "sql_error", "redis_error", "ipc_error":
		return true
	default:
		return false
	}
}

func isEndpointRPCTarget(target string) bool {
	switch target {
	case "sql", "redis", "ipc":
		return true
	default:
		return false
	}
}
