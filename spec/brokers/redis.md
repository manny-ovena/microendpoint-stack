# Redis Broker Specification

## Overview

The Redis Broker provides pooled, RPC-based access to Redis.  
Endpoints and subscribers call it via msgpack RPC.

Centralized responsibilities:

- connection pooling
- command execution
- error handling
- metrics
- logging

## RPC Methods

### GetUserCache

Parameters:

- id (int)

Returns:

- cached user record or null

### SetUserCache

Parameters:

- id (int)
- profile (object)

Returns:

- success boolean

---

## Metrics

| Metric Name               | Description                           | Dimensions                                      |
|---------------------------|---------------------------------------|-------------------------------------------------|
| redis_command_latency_ms  | Latency of Redis command execution     | service_name, redis_command, error_type         |
| redis_pool_saturation_pct | Redis pool saturation percentage       | service_name                                    |

---

## Log Fields

| Field                    | Description                               |
|--------------------------|-------------------------------------------|
| redis_command            | Redis command name                        |
| redis_latency_ms         | Redis command latency                     |
| redis_error              | Redis error or null                       |
| redis_pool_saturation_pct| Redis pool saturation                     |

---
