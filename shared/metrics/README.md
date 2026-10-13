# Shared metrics

This package is the shared Prometheus metrics API for endpoints, subscribers,
brokers, and RPC calls described in
[`spec/observability.md`](../../spec/observability.md). The registry and
endpoint, subscriber, broker, and RPC metrics are implemented.

The module uses
[`github.com/prometheus/client_golang`](https://github.com/prometheus/client_golang)
for registry and collector types. The HTTP scrape handler uses its `promhttp`
package.

The public API is organized into:

- `Registry` for per-service collector registration and scrape exposition.
- `EndpointMetrics` for HTTP request and error metric families.
- `SubscriberMetrics` for event processing, errors, and consumer lag.
- `BrokerMetrics` for SQL, Redis, and IPC metric families.
- `RPCMetrics` for RPC latency and queue-depth metric families.

Metric names and units should remain compatible with the specification,
including its millisecond latency names. Label values must use the bounded
allowed values in the spec; HTTP paths must be normalized before being used as
labels. Do not use request identifiers, raw URLs, or other unbounded values as
labels.
