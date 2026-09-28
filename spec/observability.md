# Observability

## Overview

This system uses **Prometheus** for metrics, **Loki** for logs, and **Grafana** for dashboards.  
All services (endpoints, subscribers, brokers) expose consistent, low-cardinality metrics and structured JSON logs.

Observability is designed to be:

- lightweight  
- cloud-portable  
- kind-compatible  
- on-prem compatible  
- production-grade  

---

# Metrics

All metrics follow Prometheus naming conventions:

`<service>_<metric>_<unit>`

All metrics include **low-cardinality labels** with **explicit allowed values**.

Below are the **metric dimensions and descriptions**, organized by service type.

## Endpoint Metrics

| Metric Name                | Description                               | Dimensions (Allowed Values)                                                                 |
|----------------------------|-------------------------------------------|----------------------------------------------------------------------------------------------|
| http_request_latency_ms    | Total HTTP request latency in ms          | service_name: endpoint-*; http_method: GET/POST/PUT/DELETE; http_path: normalized path; status_code: 200/400/401/403/404/409/500; rpc_target: sql/redis/ipc |
| http_requests_total        | Total number of HTTP requests             | service_name: endpoint-*; http_method: GET/POST/PUT/DELETE; http_path: normalized path       |
| http_errors_total          | Total number of HTTP errors               | service_name: endpoint-*; error_type: json_decode/rpc_error/sql_error/redis_error/ipc_error; rpc_target: sql/redis/ipc |
| endpoint_cpu_seconds_total | CPU usage                                 | service_name: endpoint-*; node_name: k8s node name                                           |
| endpoint_memory_bytes      | Memory usage                              | service_name: endpoint-*; node_name: k8s node name                                           |

## Subscriber Metrics

| Metric Name                     | Description                                   | Dimensions (Allowed Values)                                                                 |
|---------------------------------|-----------------------------------------------|----------------------------------------------------------------------------------------------|
| kafka_consumer_lag              | Kafka lag for subscriber                      | service_name: subscriber-*; kafka_topic: user.created/order.placed/inventory.changed; kafka_partition: 0–N |
| event_processing_latency_ms     | Time to process a Kafka event                 | service_name: subscriber-*; event_type: user.created/order.placed/inventory.changed; kafka_topic: same as above |
| subscriber_errors_total         | Total subscriber errors                       | service_name: subscriber-*; error_type: json_decode/rpc_error/sql_error/redis_error/ipc_error; dlq_reason: null/parse_error/rpc_failure/validation_error |
| subscriber_backlog_events_total | Total backlog events (for autoscaling)        | service_name: subscriber-*; kafka_topic: user.created/order.placed/inventory.changed         |

## Broker Metrics

### SQL Broker

| Metric Name             | Description                           | Dimensions (Allowed Values)                                           |
|-------------------------|---------------------------------------|------------------------------------------------------------------------|
| sql_query_latency_ms    | Latency of SQL query execution         | service_name: broker-sql; sql_query_name: insert_user/insert_order/update_inventory/get_user/get_order/get_inventory; error_type: null/sql_error |
| sql_pool_saturation_pct | Connection pool saturation percentage  | service_name: broker-sql                                               |

### Redis Broker

| Metric Name               | Description                           | Dimensions (Allowed Values)                                           |
|---------------------------|---------------------------------------|------------------------------------------------------------------------|
| redis_command_latency_ms  | Latency of Redis command execution     | service_name: broker-redis; redis_command: GET/SET/INCR/DECR/HGET/HSET; error_type: null/redis_error |
| redis_pool_saturation_pct | Redis pool saturation percentage       | service_name: broker-redis                                             |

### IPC Broker

| Metric Name               | Description                           | Dimensions (Allowed Values)                                           |
|---------------------------|---------------------------------------|------------------------------------------------------------------------|
| ipc_operation_latency_ms  | Latency of IPC operation               | service_name: broker-ipc; ipc_operation: read/write/delete/list; error_type: null/ipc_error |
| ipc_bytes_written_total   | Total bytes written                    | service_name: broker-ipc                                               |
| ipc_bytes_read_total      | Total bytes read                       | service_name: broker-ipc                                               |

### RPC Layer

| Metric Name        | Description                         | Dimensions (Allowed Values)                                           |
|--------------------|-------------------------------------|------------------------------------------------------------------------|
| rpc_latency_ms     | Latency of RPC call                 | service_name: endpoint-*/subscriber-*; rpc_method: InsertUser/InsertOrder/UpdateInventory/GetUser/GetOrder/GetInventory; error_type: null/rpc_error/sql_error/redis_error/ipc_error |
| rpc_queue_depth    | Depth of RPC queue                  | service_name: endpoint-*/subscriber-*                                 |

# Log Schema
All logs use **zerolog JSON**, optimized for Loki ingestion.

Below are the **log fields and descriptions**, with **explicit allowed values** where applicable.

## Core Fields (All Services)

| Field            | Description                                      | Allowed Values                                               |
|------------------|--------------------------------------------------|--------------------------------------------------------------|
| timestamp        | RFC3339 timestamp                                | RFC3339                                                      |
| service_name     | Name of the service                              | endpoint-*/ subscriber-* / broker-sql / broker-redis / broker-ipc |
| service_version  | Git SHA or semantic version                      | git SHA / semver                                             |
| environment      | Deployment environment                           | dev / kind / onprem / cloud                                  |
| pod_name         | Kubernetes pod name                              | k8s pod name                                                 |
| node_name        | Kubernetes node name                             | k8s node name                                                |
| trace_id         | Distributed trace ID                             | UUID                                                         |
| span_id          | Span ID                                          | UUID                                                         |
| correlation_id   | User-level correlation ID                        | UUID                                                         |
| log_level        | Log severity                                     | debug / info / warn / error                                  |

## Endpoint Log Fields

| Field                    | Description                               | Allowed Values                                               |
|--------------------------|-------------------------------------------|--------------------------------------------------------------|
| http_method              | HTTP method                                | GET / POST / PUT / DELETE                                   |
| http_path                | Normalized HTTP path                       | /users /orders /inventory /…                                |
| http_status              | HTTP status code                           | 200 / 400 / 401 / 403 / 404 / 409 / 500                      |
| http_latency_ms          | Total HTTP latency                         | integer                                                      |
| client_ip                | Optional anonymized client IP              | hashed IP / null                                             |
| json_decode_latency_ms   | JSON decode latency                        | integer                                                      |
| rpc_target               | Downstream broker                           | sql / redis / ipc                                            |
| rpc_latency_ms           | RPC call latency                           | integer                                                      |
| rpc_error                | RPC error message or null                  | null / rpc_error / sql_error / redis_error / ipc_error       |
| sql_query_name           | Logical SQL query name                     | insert_user / insert_order / update_inventory / get_user / get_order / get_inventory |
| sql_rows_returned        | Number of rows returned                    | integer                                                      |

## Subscriber Log Fields

| Field                      | Description                               | Allowed Values                                               |
|----------------------------|-------------------------------------------|--------------------------------------------------------------|
| kafka_topic                | Kafka topic name                          | user.created / order.placed / inventory.changed             |
| kafka_partition            | Partition number                          | 0–N                                                          |
| kafka_offset               | Offset of consumed message                | integer                                                      |
| kafka_lag                  | Lag for this partition                    | integer                                                      |
| event_type                 | Logical event type                        | user.created / order.placed / inventory.changed             |
| event_processing_latency_ms| Time to process event                     | integer                                                      |
| event_size_bytes           | Size of event payload                     | integer                                                      |
| dlq_reason                 | Dead-letter reason or null                | null / parse_error / rpc_failure / validation_error         |
| retry_count                | Number of retries                         | integer                                                      |
| rpc_target                 | Downstream broker                         | sql / redis / ipc                                            |
| rpc_latency_ms             | RPC call latency                          | integer                                                      |
| rpc_error                  | RPC error message or null                 | null / rpc_error / sql_error / redis_error / ipc_error       |

## Broker Log Fields

### SQL Broker

| Field                    | Description                               | Allowed Values                                               |
|--------------------------|-------------------------------------------|--------------------------------------------------------------|
| sql_query_name           | Logical SQL query name                    | insert_user / insert_order / update_inventory / get_user / get_order / get_inventory |
| sql_latency_ms           | SQL query latency                         | integer                                                      |
| sql_error                | SQL error or null                         | null / sql_error                                             |
| pgx_pool_saturation_pct  | Postgres pool saturation                  | 0–100                                                        |
| pgx_wait_ms              | Wait time for pool                        | integer                                                      |
| rows_returned            | Number of rows returned                   | integer                                                      |
| circuit_breaker_state    | Circuit breaker state                     | open / closed / half-open                                   |

### Redis Broker

| Field                    | Description                               | Allowed Values                                               |
|--------------------------|-------------------------------------------|--------------------------------------------------------------|
| redis_command            | Redis command name                        | GET / SET / INCR / DECR / HGET / HSET                       |
| redis_latency_ms         | Redis command latency                     | integer                                                      |
| redis_error              | Redis error or null                       | null / redis_error                                           |
| redis_pool_saturation_pct| Redis pool saturation                     | 0–100                                                        |

### IPC Broker

| Field                    | Description                               | Allowed Values                                               |
|--------------------------|-------------------------------------------|--------------------------------------------------------------|
| ipc_operation            | Operation name                            | read / write / delete / list                                |
| ipc_latency_ms           | IPC operation latency                     | integer                                                      |
| ipc_error                | IPC error or null                         | null / ipc_error                                             |
| ipc_bytes_read           | Bytes read                                | integer                                                      |
| ipc_bytes_written        | Bytes written                             | integer                                                      |

### RPC Layer

| Field                    | Description                               | Allowed Values                                               |
|--------------------------|-------------------------------------------|--------------------------------------------------------------|
| rpc_method               | RPC method name                           | InsertUser / InsertOrder / UpdateInventory / GetUser / GetOrder / GetInventory |
| rpc_latency_ms           | RPC latency                               | integer                                                      |
| rpc_error                | RPC error or null                         | null / rpc_error / sql_error / redis_error / ipc_error       |
| queue_depth              | RPC queue depth                           | integer                                                      |
| backpressure_active      | Whether backpressure is active            | true / false                                                 |
