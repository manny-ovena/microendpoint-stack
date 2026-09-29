# Endpoint Specification: orders.create

## Overview

Creates a new order record.

Stack:

- fasthttp
- simdjson-go
- msgpack RPC → SQL Broker
- zerolog

## Request

POST /orders  
{
  "user_id": int,
  "items": [...]
}

## Response

201 Created  
{
  "order_id": int
}

---

## RPC Call

SQL Broker → InsertOrder(user_id, items)

---

## Metrics

| Metric Name             | Description                           | Dimensions                                      |
|-------------------------|---------------------------------------|-------------------------------------------------|
| http_request_latency_ms | Total HTTP latency                     | service_name, http_method, http_path, status_code, rpc_target |
| rpc_latency_ms          | RPC call latency                       | service_name, rpc_method, error_type            |
| sql_query_latency_ms    | SQL query latency                      | service_name, sql_query_name, error_type        |

---

## Logs

| Field            | Description               |
|------------------|---------------------------|
| http_method      | POST                      |
| http_path        | /orders                   |
| rpc_target       | sql                       |
| rpc_latency_ms   | RPC latency               |
| sql_query_name   | insert_order              |

---
