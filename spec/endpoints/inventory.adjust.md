# Endpoint Specification: inventory.adjust

## Overview

Adjusts inventory for a SKU.

Stack:

- fasthttp
- simdjson-go
- msgpack RPC → SQL Broker
- zerolog

## Request

POST /inventory/adjust  
{
  "sku": string,
  "delta": int
}

## Response

200 OK  
{
  "sku": string,
  "new_count": int
}

---

## RPC Call

SQL Broker → UpdateInventory(sku, delta)

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
| http_path        | /inventory/adjust         |
| rpc_target       | sql                       |
| rpc_latency_ms   | RPC latency               |
| sql_query_name   | update_inventory          |

---
