package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry owns the Prometheus collectors for one service.
type Registry struct {
	registry *prometheus.Registry
}

// NewRegistry creates a registry for a service.
func NewRegistry() (*Registry, error) {
	return &Registry{registry: prometheus.NewRegistry()}, nil
}

// Register adds a collector to the service registry.
func (r *Registry) Register(c prometheus.Collector) error {
	if r == nil || r.registry == nil {
		return fmt.Errorf("registry is not initialized")
	}
	return r.registry.Register(c)
}

// Handler exposes the service registry in Prometheus' scrape format.
func (r *Registry) Handler() (http.Handler, error) {
	if r == nil || r.registry == nil {
		return nil, fmt.Errorf("registry is not initialized")
	}
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{}), nil
}
