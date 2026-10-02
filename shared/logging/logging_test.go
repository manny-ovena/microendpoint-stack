package logging

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestNewWritesSchemaFieldsAsJSON(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(Config{
		ServiceName:    "endpoint-users-get",
		ServiceVersion: "abc123",
		Environment:    "kind",
		PodName:        "users-get-abc",
		NodeName:       "kind-worker",
		Level:          "debug",
		Output:         &output,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Debug().Msg("request complete")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}
	fields := map[string]string{
		"service_name":    "endpoint-users-get",
		"service_version": "abc123",
		"environment":     "kind",
		"pod_name":        "users-get-abc",
		"node_name":       "kind-worker",
		"log_level":       "debug",
		"message":         "request complete",
	}
	for key, want := range fields {
		if got, ok := entry[key].(string); !ok || got != want {
			t.Errorf("%s = %v, want %q", key, entry[key], want)
		}
	}
	timestamp, ok := entry["timestamp"].(string)
	if !ok {
		t.Fatal("timestamp field missing or not a string")
	}
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Errorf("timestamp %q is not RFC3339: %v", timestamp, err)
	}
}

func TestWithTraceContextAddsNonEmptyFields(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(Config{ServiceName: "subscriber-user-created", Output: &output})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	loggerWithTrace := WithTraceContext(logger, "trace-1", "span-2", "correlation-3")
	loggerWithTrace.Info().Msg("event handled")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}
	for key, want := range map[string]string{
		"trace_id":       "trace-1",
		"span_id":        "span-2",
		"correlation_id": "correlation-3",
	} {
		if got := entry[key]; got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
}

func TestNewRequiresServiceNameAndValidLevel(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("New() without service name succeeded")
	}
	if _, err := New(Config{ServiceName: "endpoint-users-get", Level: "verbose"}); err == nil {
		t.Error("New() with invalid log level succeeded")
	}
	if _, err := New(Config{ServiceName: "endpoint-users-get", Environment: "staging"}); err == nil {
		t.Error("New() with invalid environment succeeded")
	}
}
