# Subscriber Specification: order.placed

## Overview

The `order.placed` subscriber processes events emitted when an order is placed.

Stack:

- kafka-go
- simdjson-go
- msgpack RPC → SQL Broker
- zerolog

## Event Schema

{
  "order_id": int,
  "user_id": int,
  "items": [...]
}

## Processing Flow

1. Kafka event arrives
2. simdjson-go parses event
3. msgpack RPC → SQL Broker (InsertOrder)
4. SQL Broker inserts order row
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
| event_type                 | order.placed                              |
| rpc_target                 | sql                                        |
| rpc_latency_ms             | RPC latency                               |
| sql_query_name             | insert_order                              |

---
