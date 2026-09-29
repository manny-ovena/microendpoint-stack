# SQL Broker Specification

## Overview

The SQL Broker is a thin RPC service that provides safe, pooled, prepared-statement access to Postgres.  
Endpoints and subscribers never talk to Postgres directly — they call this broker via msgpack RPC.

The SQL Broker centralizes:

- connection pooling
- prepared statements
- error handling
- metrics
- logging
- retry/backoff

## RPC Methods

### GetUser

Parameters:

- id (int)

Returns:

- user record (msgpack)

### InsertUser

Parameters:

- id (int)
- email (string)

Returns:

- success boolean

### UpdateInventory

Parameters:

- sku (string)
- delta (int)

Returns:

- updated inventory count

---

## Metrics

| Metric Name             | Description                           | Dimensions                                      |
|-------------------------|---------------------------------------|-------------------------------------------------|
| sql_query_latency_ms    | Latency of SQL query execution         | service_name, sql_query_name, error_type        |
| sql_pool_saturation_pct | Connection pool saturation percentage  | service_name                                    |

---

## Log Fields

| Field                    | Description                               |
|--------------------------|-------------------------------------------|
| sql_query_name           | Logical SQL query name                    |
| sql_latency_ms           | SQL query latency                         |
| sql_error                | SQL error or null                         |
| pgx_pool_saturation_pct  | Postgres pool saturation                  |
| pgx_wait_ms              | Wait time for pool                        |
| rows_returned            | Number of rows returned                   |

---
