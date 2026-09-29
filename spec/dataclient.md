# DataClient Specification

## Overview

The DataClient is the unified RPC client used by endpoints and subscribers to communicate with the SQL, Redis, and IPC brokers.

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

### IPC Broker Methods

| Method Name | Description |
|-------------|-------------|
| FileRead    | Read file contents |
| FileWrite   | Write file contents |
| FileDelete  | Delete file |
| FileList    | List directory contents |

#### Allowed Values

- ipc_operation: read / write / delete / list
- error_type: null / ipc_error / rpc_error

## Error Semantics

All errors are normalized into:

error: "<string|null>"

### Error Types

| Error Type   | Description |
|--------------|-------------|
| rpc_error    | Transport or protocol failure |
| sql_error    | SQL broker returned error |
| redis_error  | Redis broker returned error |
| ipc_error    | IPC broker returned error |
| timeout      | RPC timeout exceeded |
| backpressure | RPC queue depth exceeded threshold |

#### Allowed Values

- error_type: rpc_error / sql_error / redis_error / ipc_error / timeout / backpressure
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
| rpc_latency_ms  | RPC latency | service_name: endpoint-*/ subscriber-*; rpc_method: InsertUser/InsertOrder/UpdateInventory/GetUser/GetOrder/GetInventory/CacheGet/CacheSet/CacheIncr/CacheDecr/CacheHashGet/CacheHashSet/FileRead/FileWrite/FileDelete/FileList; error_type: null/rpc_error/sql_error/redis_error/ipc_error |
| rpc_queue_depth | Queue depth | service_name: endpoint-*/ subscriber-* |

## Logging

### Log Fields

| Field              | Description | Allowed Values |
|--------------------|-------------|----------------|
| rpc_method         | RPC method name | InsertUser / InsertOrder / UpdateInventory / GetUser / GetOrder / GetInventory / CacheGet / CacheSet / CacheIncr / CacheDecr / CacheHashGet / CacheHashSet / FileRead / FileWrite / FileDelete / FileList |
| rpc_latency_ms     | RPC latency | integer |
| rpc_error          | RPC error | null / rpc_error / sql_error / redis_error / ipc_error |
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
