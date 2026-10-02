package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	// Zerolog keeps these schema-wide JSON settings as package globals.
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = time.RFC3339
}

// Config contains the shared metadata written to every log event.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	PodName        string
	NodeName       string
	Level          string
	Output         io.Writer
}

// New creates a JSON logger with the core fields required by the observability
// schema. Empty metadata values are read from their corresponding environment
// variables and then defaulted where appropriate.
func New(cfg Config) (zerolog.Logger, error) {
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		return zerolog.Logger{}, fmt.Errorf("service name is required")
	}

	serviceVersion := valueOrEnv(cfg.ServiceVersion, "SERVICE_VERSION", "GIT_SHA")
	if serviceVersion == "" {
		serviceVersion = "dev"
	}
	environment := valueOrEnv(cfg.Environment, "ENVIRONMENT")
	if environment == "" {
		environment = "dev"
	}
	switch environment {
	case "dev", "kind", "onprem", "cloud":
	default:
		return zerolog.Logger{}, fmt.Errorf("invalid environment %q", environment)
	}
	podName := valueOrEnv(cfg.PodName, "POD_NAME")
	nodeName := valueOrEnv(cfg.NodeName, "NODE_NAME")

	levelName := strings.ToLower(strings.TrimSpace(cfg.Level))
	if levelName == "" {
		levelName = strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	}
	if levelName == "" {
		levelName = zerolog.InfoLevel.String()
	}
	if levelName == "warning" {
		levelName = "warn"
	}
	switch levelName {
	case "debug", "info", "warn", "error":
	default:
		return zerolog.Logger{}, fmt.Errorf("invalid log level %q: want debug, info, warn, or error", levelName)
	}
	level, err := zerolog.ParseLevel(levelName)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("invalid log level %q: %w", levelName, err)
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	context := zerolog.New(output).Level(level).With().
		Timestamp().
		Str("service_name", serviceName).
		Str("service_version", serviceVersion).
		Str("environment", environment)
	if podName != "" {
		context = context.Str("pod_name", podName)
	}
	if nodeName != "" {
		context = context.Str("node_name", nodeName)
	}

	return context.Logger().Hook(logLevelHook{}), nil
}

// NewFromEnv creates a logger for serviceName using the standard environment
// variables documented by this package.
func NewFromEnv(serviceName string) (zerolog.Logger, error) {
	return New(Config{ServiceName: serviceName})
}

// WithTraceContext adds any supplied trace identifiers to a logger. Empty
// values are omitted so logs do not contain misleading empty identifiers.
func WithTraceContext(logger zerolog.Logger, traceID, spanID, correlationID string) zerolog.Logger {
	context := logger.With()
	if traceID != "" {
		context = context.Str("trace_id", traceID)
	}
	if spanID != "" {
		context = context.Str("span_id", spanID)
	}
	if correlationID != "" {
		context = context.Str("correlation_id", correlationID)
	}
	return context.Logger()
}

func valueOrEnv(value string, environmentVariables ...string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	for _, name := range environmentVariables {
		if value = strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

type logLevelHook struct{}

func (logLevelHook) Run(event *zerolog.Event, level zerolog.Level, _ string) {
	event.Str("log_level", level.String())
}
