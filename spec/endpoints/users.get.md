# Endpoint Specification: users.get

## Overview

The `users.get` endpoint retrieves a user record by ID.  
It is optimized for extremely low latency and high throughput.

Stack:

- fasthttp
- simdjson-go
- msgpack RPC → SQL Broker
- zerolog

## Request

GET /users?id=&lt;int&gt;

## Response

200 OK  
{
  "id": int,
  "email": string
}

---

## RPC Call

SQL Broker → GetUser(id)

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
| http_method      | GET                       |
| http_path        | /users                    |
| rpc_target       | sql                       |
| rpc_latency_ms   | RPC latency               |
| sql_query_name   | get_user                  |

---
