# DataClient Specification

## Overview

The DataClient is the unified RPC client used by endpoints and subscribers to
communicate with the SQL and Redis brokers and the Kafka Publishing Broker.
IPC methods are deferred until a concrete, portable filesystem use case is
established.

It provides:

- msgpack serialization
- TCP transport
- request/response RPC
- consistent error semantics
- consistent metrics
- consistent logging
- consistent retry/backoff behavior

The DataClient behaves identically across:

- kind
- on‑prem (native kubeadm Kubernetes)
- cloud (AKS/EKS/GKE)

Only broker endpoints differ by environment.

## RPC Protocol

### Request Format

{
  "method": "<string>",
  "params": "<object>",
  "trace_id": "<uuid>",
  "span_id": "<uuid>"
}

### Response Format

{
  "ok": "<bool>",
  "result": "<object>",
  "error": "<string|null>"
}

## Supported RPC Methods

### SQL Broker Methods

| Method Name     | Description |
|-----------------|-------------|
| InsertUser      | Insert a new user row |
| InsertOrder     | Insert a new order row |
| UpdateInventory | Update inventory quantity |
| GetUser         | Fetch user by ID |
| GetOrder        | Fetch order by ID |
| GetInventory    | Fetch inventory item by ID |

#### Allowed Values

- sql_query_name: insert_user / insert_order / update_inventory / get_user / get_order / get_inventory
- error_type: null / sql_error / rpc_error

### Redis Broker Methods

| Method Name    | Description |
|----------------|-------------|
| CacheGet       | GET key |
| CacheSet       | SET key |
| CacheIncr      | INCR key |
| CacheDecr      | DECR key |
| CacheHashGet   | HGET field |
| CacheHashSet   | HSET field |

#### Allowed Values

- redis_command: GET / SET / INCR / DECR / HGET / HSET
- error_type: null / redis_error / rpc_error

### Kafka Publishing Broker Methods

| Method Name | Description |
|-------------|-------------|
| Publish     | Publish a message to Kafka through the publishing broker |

#### Allowed Values

- error_type: null / kafka_error / rpc_error

The publishing broker centralizes Kafka-facing producer connections. Endpoint
replicas call the broker over RPC, while the broker reuses long-lived producer
clients and has a bounded replica count. This keeps Kafka-facing connection
growth independent of endpoint replica growth.

## Error Semantics

All errors are normalized into:

error: "<string|null>"

### Error Types

| Error Type   | Description |
|--------------|-------------|
| rpc_error    | Transport or protocol failure |
| sql_error    | SQL broker returned error |
| redis_error  | Redis broker returned error |
| kafka_error  | Kafka publishing broker returned error |
| timeout      | RPC timeout exceeded |
| backpressure | RPC queue depth exceeded threshold |

#### Allowed Values

- error_type: rpc_error / sql_error / redis_error / kafka_error / timeout / backpressure
- error_message: free‑form string

## Retry & Backoff

The DataClient implements:

- exponential backoff
- jitter
- max retry count
- timeout enforcement

### Configurable Fields

| Field                   | Description | Allowed Values |
|-------------------------|-------------|----------------|
| RPC_TIMEOUT_MS          | RPC timeout | integer |
| RPC_MAX_RETRIES         | Retry count | integer |
| RPC_BACKOFF_MS          | Initial backoff | integer |
| RPC_BACKPRESSURE_THRESHOLD | Max queue depth | integer |

## Metrics

### RPC Metrics

| Metric Name     | Description | Dimensions (Allowed Values) |
|-----------------|-------------|------------------------------|
| rpc_latency_ms  | RPC latency | service_name: endpoint-*/ subscriber-*; rpc_method: InsertUser/InsertOrder/UpdateInventory/GetUser/GetOrder/GetInventory/CacheGet/CacheSet/CacheIncr/CacheDecr/CacheHashGet/CacheHashSet/Publish; error_type: null/rpc_error/sql_error/redis_error/kafka_error |
| rpc_queue_depth | Queue depth | service_name: endpoint-*/ subscriber-* |

## Logging

### Log Fields

| Field              | Description | Allowed Values |
|--------------------|-------------|----------------|
| rpc_method         | RPC method name | InsertUser / InsertOrder / UpdateInventory / GetUser / GetOrder / GetInventory / CacheGet / CacheSet / CacheIncr / CacheDecr / CacheHashGet / CacheHashSet / Publish |
| rpc_latency_ms     | RPC latency | integer |
| rpc_error          | RPC error | null / rpc_error / sql_error / redis_error / kafka_error |
| queue_depth        | RPC queue depth | integer |
| backpressure_active| Backpressure flag | true / false |
| trace_id           | Trace ID | UUID |
| span_id            | Span ID | UUID |

## Connection Management

The DataClient maintains:

- persistent TCP connections
- connection pooling
- automatic reconnect
- health checks

The Kafka Publishing Broker maintains reusable producer connections. Its
replica count should be bounded to control Kafka-facing connections
independently of endpoint replica count.

### Allowed Values

| Field               | Description | Allowed Values |
|---------------------|-------------|----------------|
| connection_state    | Current state | connected / disconnected / reconnecting |
| connection_errors_total | Total connection errors | integer |

## Environment Behavior

The DataClient behaves identically across:

- kind
- on‑prem
- cloud

There are no environment‑specific differences in:

- RPC protocol
- error semantics
- metrics
- logging
- retry/backoff
- connection pooling

Only broker endpoints differ:

| Environment | Example Endpoint |
| ------------ | ------------------ |
| kind | broker-sql.kind.svc.cluster.local:9000 |
| on‑prem | broker-sql.onprem.svc.cluster.local:9000 |
| cloud | broker-sql.cloud.svc.cluster.local:9000 |

The Kafka publishing endpoint is configured separately using
`KAFKA_PUBLISHER_BROKER_ADDR`. The scaffold does not prescribe its port or
deploy a service endpoint yet. SQL and Redis endpoints use `SQL_BROKER_ADDR`
and `REDIS_BROKER_ADDR`, respectively.
