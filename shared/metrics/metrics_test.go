package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestRegistryRegistersCollectorsAndExposesMetrics(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}

	collector := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "registry_test_gauge",
		Help: "A registry test gauge.",
	})
	collector.Set(7)
	if err := registry.Register(collector); err != nil {
		t.Fatal(err)
	}

	handler, err := registry.Handler()
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("handler status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "registry_test_gauge 7") {
		t.Fatalf("handler response does not expose registered gauge: %s", response.Body.String())
	}
}

func TestEndpointMetricsMatchObservabilitySchema(t *testing.T) {
	registry := newTestRegistry(t)
	metrics, err := NewEndpointMetrics(registry, Config{ServiceName: "endpoint-users-get"})
	if err != nil {
		t.Fatal(err)
	}

	if err := metrics.ObserveHTTPRequest("GET", "/users", 200, "sql", 1250*time.Microsecond); err != nil {
		t.Fatal(err)
	}
	if err := metrics.ObserveHTTPRequest("GET", "/users", 200, "sql", 750*time.Microsecond); err != nil {
		t.Fatal(err)
	}
	if err := metrics.IncrementHTTPError("sql_error", "sql"); err != nil {
		t.Fatal(err)
	}
	if err := metrics.AddCPUTime("users-get-pod", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := metrics.SetMemoryUsage("users-get-pod", 1024); err != nil {
		t.Fatal(err)
	}

	families := gatherTestFamilies(t, registry)
	assertFamilyNames(t, families,
		"http_request_latency_ms",
		"http_requests_total",
		"http_errors_total",
		"endpoint_cpu_seconds_total",
		"endpoint_memory_bytes",
	)
	assertMetricLabels(t, onlyMetric(t, families, "http_request_latency_ms"),
		map[string]string{
			"service_name": "endpoint-users-get",
			"http_method":  "GET",
			"http_path":    "/users",
			"status_code":  "200",
			"rpc_target":   "sql",
		})
	histogram := onlyMetric(t, families, "http_request_latency_ms").GetHistogram()
	if histogram.GetSampleCount() != 2 {
		t.Errorf("HTTP latency sample count = %d, want 2", histogram.GetSampleCount())
	}
	if got, want := histogram.GetSampleSum(), 2.0; got != want {
		t.Errorf("HTTP latency sum = %v ms, want %v ms", got, want)
	}

	requests := onlyMetric(t, families, "http_requests_total")
	assertMetricLabels(t, requests, map[string]string{
		"service_name": "endpoint-users-get",
		"http_method":  "GET",
		"http_path":    "/users",
	})
	if got := requests.GetCounter().GetValue(); got != 2 {
		t.Errorf("HTTP request count = %v, want 2", got)
	}

	httpErrors := onlyMetric(t, families, "http_errors_total")
	assertMetricLabels(t, httpErrors, map[string]string{
		"service_name": "endpoint-users-get",
		"error_type":   "sql_error",
		"rpc_target":   "sql",
	})
	if got := httpErrors.GetCounter().GetValue(); got != 1 {
		t.Errorf("HTTP error count = %v, want 1", got)
	}

	cpu := onlyMetric(t, families, "endpoint_cpu_seconds_total")
	assertMetricLabels(t, cpu, map[string]string{
		"service_name": "endpoint-users-get",
		"pod_name":     "users-get-pod",
	})
	if got := cpu.GetCounter().GetValue(); got != 1 {
		t.Errorf("endpoint CPU time = %v seconds, want 1", got)
	}

	memory := onlyMetric(t, families, "endpoint_memory_bytes")
	assertMetricLabels(t, memory, map[string]string{
		"service_name": "endpoint-users-get",
		"pod_name":     "users-get-pod",
	})
	if got := memory.GetGauge().GetValue(); got != 1024 {
		t.Errorf("endpoint memory usage = %v bytes, want 1024", got)
	}
}

func TestEndpointMetricsRejectOutOfSchemaLabels(t *testing.T) {
	metrics, err := NewEndpointMetrics(newTestRegistry(t), Config{ServiceName: "endpoint-users-get"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{"method", func() error { return metrics.ObserveHTTPRequest("PATCH", "/users", 200, "sql", time.Second) }},
		{"path", func() error { return metrics.ObserveHTTPRequest("GET", "", 200, "sql", time.Second) }},
		{"status code", func() error { return metrics.ObserveHTTPRequest("GET", "/users", 201, "sql", time.Second) }},
		{"request RPC target", func() error { return metrics.ObserveHTTPRequest("GET", "/users", 200, "kafka", time.Second) }},
		{"error type", func() error { return metrics.IncrementHTTPError("database_failure", "sql") }},
		{"error RPC target", func() error { return metrics.IncrementHTTPError("sql_error", "kafka") }},
		{"negative request duration", func() error { return metrics.ObserveHTTPRequest("GET", "/users", 200, "sql", -time.Second) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err == nil {
				t.Fatal("expected invalid metric input to be rejected")
			}
		})
	}
}

func TestSubscriberMetricsMatchObservabilitySchema(t *testing.T) {
	registry := newTestRegistry(t)
	metrics, err := NewSubscriberMetrics(registry, Config{ServiceName: "subscriber-user-created"})
	if err != nil {
		t.Fatal(err)
	}

	if err := metrics.ObserveEventProcessing("user.created", "user.created", 1500*time.Microsecond); err != nil {
		t.Fatal(err)
	}
	if err := metrics.IncrementError("rpc_error", "rpc_failure"); err != nil {
		t.Fatal(err)
	}
	if err := metrics.SetConsumerLag("user.created", 2, 12); err != nil {
		t.Fatal(err)
	}
	if err := metrics.AddBacklogEvents("user.created", 3); err != nil {
		t.Fatal(err)
	}

	families := gatherTestFamilies(t, registry)
	assertFamilyNames(t, families,
		"kafka_consumer_lag",
		"event_processing_latency_ms",
		"subscriber_errors_total",
		"subscriber_backlog_events_total",
	)
	latency := onlyMetric(t, families, "event_processing_latency_ms")
	assertMetricLabels(t, latency, map[string]string{
		"service_name": "subscriber-user-created",
		"event_type":   "user.created",
		"kafka_topic":  "user.created",
	})
	if got := latency.GetHistogram().GetSampleSum(); got != 1.5 {
		t.Errorf("event processing latency sum = %v ms, want 1.5", got)
	}

	errors := onlyMetric(t, families, "subscriber_errors_total")
	assertMetricLabels(t, errors, map[string]string{
		"service_name": "subscriber-user-created",
		"error_type":   "rpc_error",
		"dlq_reason":   "rpc_failure",
	})
	if got := errors.GetCounter().GetValue(); got != 1 {
		t.Errorf("subscriber error count = %v, want 1", got)
	}

	lag := onlyMetric(t, families, "kafka_consumer_lag")
	assertMetricLabels(t, lag, map[string]string{
		"service_name":    "subscriber-user-created",
		"kafka_topic":     "user.created",
		"kafka_partition": "2",
	})
	if got := lag.GetGauge().GetValue(); got != 12 {
		t.Errorf("Kafka consumer lag = %v, want 12", got)
	}

	backlog := onlyMetric(t, families, "subscriber_backlog_events_total")
	assertMetricLabels(t, backlog, map[string]string{
		"service_name": "subscriber-user-created",
		"kafka_topic":  "user.created",
	})
	if got := backlog.GetCounter().GetValue(); got != 3 {
		t.Errorf("subscriber backlog = %v, want 3", got)
	}
}

func TestBrokerMetricsMatchObservabilitySchema(t *testing.T) {
	t.Run("SQL", func(t *testing.T) {
		registry := newTestRegistry(t)
		metrics, err := NewBrokerMetrics(registry, Config{ServiceName: "broker-sql"})
		if err != nil {
			t.Fatal(err)
		}
		if err := metrics.ObserveSQLQuery("get_user", "null", 2500*time.Microsecond); err != nil {
			t.Fatal(err)
		}
		if err := metrics.SetSQLPoolSaturation(75); err != nil {
			t.Fatal(err)
		}

		families := gatherTestFamilies(t, registry)
		assertFamilyNames(t, families, "sql_query_latency_ms", "sql_pool_saturation_pct")
		query := onlyMetric(t, families, "sql_query_latency_ms")
		assertMetricLabels(t, query, map[string]string{
			"service_name":   "broker-sql",
			"sql_query_name": "get_user",
			"error_type":     "null",
		})
		if got := query.GetHistogram().GetSampleSum(); got != 2.5 {
			t.Errorf("SQL query latency sum = %v ms, want 2.5", got)
		}
		pool := onlyMetric(t, families, "sql_pool_saturation_pct")
		assertMetricLabels(t, pool, map[string]string{"service_name": "broker-sql"})
		if got := pool.GetGauge().GetValue(); got != 75 {
			t.Errorf("SQL pool saturation = %v%%, want 75%%", got)
		}
	})

	t.Run("Redis", func(t *testing.T) {
		registry := newTestRegistry(t)
		metrics, err := NewBrokerMetrics(registry, Config{ServiceName: "broker-redis"})
		if err != nil {
			t.Fatal(err)
		}
		if err := metrics.ObserveRedisCommand("GET", "null", 3*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		if err := metrics.SetRedisPoolSaturation(40); err != nil {
			t.Fatal(err)
		}

		families := gatherTestFamilies(t, registry)
		assertFamilyNames(t, families, "redis_command_latency_ms", "redis_pool_saturation_pct")
		command := onlyMetric(t, families, "redis_command_latency_ms")
		assertMetricLabels(t, command, map[string]string{
			"service_name":  "broker-redis",
			"redis_command": "GET",
			"error_type":    "null",
		})
		if got := command.GetHistogram().GetSampleSum(); got != 3 {
			t.Errorf("Redis command latency sum = %v ms, want 3", got)
		}
		pool := onlyMetric(t, families, "redis_pool_saturation_pct")
		assertMetricLabels(t, pool, map[string]string{"service_name": "broker-redis"})
		if got := pool.GetGauge().GetValue(); got != 40 {
			t.Errorf("Redis pool saturation = %v%%, want 40%%", got)
		}
	})

	t.Run("IPC", func(t *testing.T) {
		registry := newTestRegistry(t)
		metrics, err := NewBrokerMetrics(registry, Config{ServiceName: "broker-ipc"})
		if err != nil {
			t.Fatal(err)
		}
		if err := metrics.ObserveIPCOperation("write", "ipc_error", 4*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		if err := metrics.AddIPCBytesWritten(64); err != nil {
			t.Fatal(err)
		}
		if err := metrics.AddIPCBytesRead(32); err != nil {
			t.Fatal(err)
		}

		families := gatherTestFamilies(t, registry)
		assertFamilyNames(t, families, "ipc_operation_latency_ms", "ipc_bytes_written_total", "ipc_bytes_read_total")
		operation := onlyMetric(t, families, "ipc_operation_latency_ms")
		assertMetricLabels(t, operation, map[string]string{
			"service_name":  "broker-ipc",
			"ipc_operation": "write",
			"error_type":    "ipc_error",
		})
		if got := operation.GetHistogram().GetSampleSum(); got != 4 {
			t.Errorf("IPC operation latency sum = %v ms, want 4", got)
		}
		written := onlyMetric(t, families, "ipc_bytes_written_total")
		assertMetricLabels(t, written, map[string]string{"service_name": "broker-ipc"})
		if got := written.GetCounter().GetValue(); got != 64 {
			t.Errorf("IPC bytes written = %v, want 64", got)
		}
		read := onlyMetric(t, families, "ipc_bytes_read_total")
		assertMetricLabels(t, read, map[string]string{"service_name": "broker-ipc"})
		if got := read.GetCounter().GetValue(); got != 32 {
			t.Errorf("IPC bytes read = %v, want 32", got)
		}
	})
}

func TestRPCMetricsMatchObservabilitySchema(t *testing.T) {
	for _, serviceName := range []string{"endpoint-users-get", "subscriber-user-created"} {
		t.Run(serviceName, func(t *testing.T) {
			registry := newTestRegistry(t)
			metrics, err := NewRPCMetrics(registry, Config{ServiceName: serviceName})
			if err != nil {
				t.Fatal(err)
			}
			if err := metrics.ObserveRPCCall("GetUser", "sql_error", 1750*time.Microsecond); err != nil {
				t.Fatal(err)
			}
			if err := metrics.SetQueueDepth(6); err != nil {
				t.Fatal(err)
			}

			families := gatherTestFamilies(t, registry)
			assertFamilyNames(t, families, "rpc_latency_ms", "rpc_queue_depth")
			latency := onlyMetric(t, families, "rpc_latency_ms")
			assertMetricLabels(t, latency, map[string]string{
				"service_name": serviceName,
				"rpc_method":   "GetUser",
				"error_type":   "sql_error",
			})
			if got := latency.GetHistogram().GetSampleSum(); got != 1.75 {
				t.Errorf("RPC latency sum = %v ms, want 1.75", got)
			}
			queue := onlyMetric(t, families, "rpc_queue_depth")
			assertMetricLabels(t, queue, map[string]string{"service_name": serviceName})
			if got := queue.GetGauge().GetValue(); got != 6 {
				t.Errorf("RPC queue depth = %v, want 6", got)
			}
		})
	}
}

func TestRPCMetricsRejectOutOfSchemaValues(t *testing.T) {
	registry := newTestRegistry(t)
	metrics, err := NewRPCMetrics(registry, Config{ServiceName: "endpoint-users-get"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewRPCMetrics(registry, Config{ServiceName: "broker-sql"}); err == nil {
		t.Error("NewRPCMetrics() accepted a non-RPC service name")
	}
	if err := metrics.ObserveRPCCall("UnknownMethod", "null", time.Millisecond); err == nil {
		t.Error("ObserveRPCCall() accepted an unknown method")
	}
	if err := metrics.ObserveRPCCall("GetUser", "unknown_error", time.Millisecond); err == nil {
		t.Error("ObserveRPCCall() accepted an unknown error type")
	}
	if err := metrics.ObserveRPCCall("GetUser", "null", -time.Millisecond); err == nil {
		t.Error("ObserveRPCCall() accepted a negative duration")
	}
	if err := metrics.SetQueueDepth(-1); err == nil {
		t.Error("SetQueueDepth() accepted a negative queue depth")
	}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	return &Registry{registry: prometheus.NewRegistry()}
}

func gatherTestFamilies(t *testing.T, registry *Registry) map[string]*dto.MetricFamily {
	t.Helper()
	families, err := registry.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		result[family.GetName()] = family
	}
	return result
}

func assertFamilyNames(t *testing.T, families map[string]*dto.MetricFamily, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := families[name]; !ok {
			t.Errorf("metric family %q was not registered", name)
		}
	}
}

func onlyMetric(t *testing.T, families map[string]*dto.MetricFamily, name string) *dto.Metric {
	t.Helper()
	family, ok := families[name]
	if !ok {
		t.Fatalf("metric family %q was not registered", name)
	}
	if got := len(family.GetMetric()); got != 1 {
		t.Fatalf("metric family %q has %d samples, want 1", name, got)
	}
	return family.GetMetric()[0]
}

func assertMetricLabels(t *testing.T, metric *dto.Metric, want map[string]string) {
	t.Helper()
	got := make(map[string]string, len(metric.GetLabel()))
	for _, label := range metric.GetLabel() {
		got[label.GetName()] = label.GetValue()
	}
	if len(got) != len(want) {
		t.Errorf("metric labels = %v, want %v", got, want)
		return
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("metric label %q = %q, want %q", name, got[name], value)
		}
	}
}
