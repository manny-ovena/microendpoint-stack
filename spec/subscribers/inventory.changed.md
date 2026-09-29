# Subscriber Specification: inventory.changed

## Overview

The `inventory.changed` subscriber processes events emitted when inventory changes.

Stack:

- kafka-go
- simdjson-go
- msgpack RPC → SQL Broker
- zerolog

## Event Schema

{
  "sku": string,
  "delta": int
}

## Processing Flow

1. Kafka event arrives
2. simdjson-go parses event
3. msgpack RPC → SQL Broker (UpdateInventory)
4. SQL Broker updates inventory
5. Subscriber logs + emits metrics

---

## Metrics

| Metric Name                   | Description                               | Dimensions                                      |
|-------------------------------|-------------------------------------------|-------------------------------------------------|
| event_processing_latency_ms   | Time to process event                     | service_name, event_type, kafka_topic           |
| kafka_consumer_lag            | Kafka lag                                 | service_name, kafka_topic, kafka_partition      |
| rpc_latency_ms                | RPC call latency                          | service_name, rpc_method, error_type            |
| sql_query_latency_ms          | SQL query latency                         | service_name, sql_query_name, error_type        |

---

## Logs

| Field                      | Description                               |
|----------------------------|-------------------------------------------|
| kafka_topic                | Kafka topic name                          |
| kafka_partition            | Partition number                          |
| kafka_offset               | Offset consumed                           |
| kafka_lag                  | Lag for this partition                    |
| event_type                 | inventory.changed                         |
| rpc_target                 | sql                                        |
| rpc_latency_ms             | RPC latency                               |
| sql_query_name             | update_inventory                          |

---
