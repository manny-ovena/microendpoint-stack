# Kafka publishing broker

This service is the application-facing publishing facade for the Kafka
cluster deployed by `k8s/base/kafka.yaml`. Callers are intended to reach it
through the shared DataClient's MessagePack RPC `Publish` method rather than
opening Kafka producer connections in every endpoint process.

`New` accepts comma-separated broker addresses through `Config.Brokers`, or
`NewFromEnv` reads them from `KAFKA_BROKERS`. The executable listens on
`KAFKA_PUBLISHER_ADDR` (default `:9000`) for MessagePack TCP RPC. It accepts
`Publish` requests using the shared `{method, params, trace_id, span_id}`
request and `{ok, result, error}` response envelope. Each TCP connection can
carry multiple sequential request/response objects.

The service creates one reusable `kafka-go` writer per process. `Publish`
synchronously waits for acknowledgments from all in-sync replicas and supports
per-message `timeout_ms`. The effective durability depends on the topic's
replication factor.

Call `Close` during service shutdown to flush pending writes and close the
writer. Keep publisher replica counts bounded to control Kafka-facing
connections independently from endpoint replica counts.

Build and run tests with `make build test vet`; build the container from the
repository root context with `make image`. At runtime, set `KAFKA_BROKERS` to a
comma-separated Kafka bootstrap list.
