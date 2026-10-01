# Architecture

## Overview

This system defines a production-grade, cloud-portable architecture for **high‑traffic Go micro‑endpoints** and **high‑throughput Go micro‑subscribers**, each communicating with a **thin data-access layer** via **binary RPC (msgpack)**.

The design goals:

- Extremely low latency  
- High throughput  
- Small containers  
- Minimal dependencies  
- Predictable autoscaling  
- Strong observability  
- Environment portability (kind → on‑prem → cloud)

The architecture is intentionally simple, explicit, and easy to reason about.

---

## Components

### MethodIngress Custom Resource

The Kubernetes deployment includes a project-defined, namespaced
`MethodIngress` CRD (`networking.microendpoints.ovena.io/v1alpha1`). A resource
references a standard Ingress by name and describes path, HTTP method, and
backend Service name/port rules. The CRD schema and example are documented in
[`shared/methodingress-controller`](../shared/methodingress-controller/README.md).

The controller generates ingress-nginx server snippets to route exact paths
by HTTP method, allowing different services to share a path. See the controller
README for ingress prerequisites, supported backends, and limitations.

### Micro‑Endpoints

Micro‑endpoints are tiny HTTP services built with:

- fasthttp (ultra-fast HTTP server)
- simdjson-go (SIMD-accelerated JSON parsing)
- zerolog (zero-allocation structured logging)
- msgpack RPC (binary RPC to brokers)

Endpoints **never** access datastores directly.  
They call brokers via RPC.

### Micro‑Subscribers

Micro‑subscribers are tiny Kafka consumers built with:

- kafka-go
- simdjson-go
- zerolog
- msgpack RPC

Subscribers scale based on **Kafka lag** using **KEDA**.

### Thin Brokers (Data-Access Layer)

Three small RPC services:

- SQL Broker → Postgres  
- Redis Broker → Redis  
- IPC Broker → filesystem / local IPC  

Brokers centralize:

- connection pooling  
- prepared statements  
- error handling  
- metrics  
- logging  

### Shared Libraries

- DataClient (msgpack RPC client + routing)
- logging (zerolog setup)
- metrics (Prometheus registry)
- config (env + defaults)

---

## Technical Rationale

### Why micro‑endpoints + micro‑subscribers?

- Independent scaling  
- Independent failure domains  
- Minimal blast radius  
- Predictable performance  
- Easier debugging  
- Easier testing  

**Alternatives considered:**  
Monolith → harder to scale, harder to isolate failures  
Combined API + worker → mixes concerns, harder to tune

---

### Why fasthttp instead of net/http?

- Lower latency  
- Fewer allocations  
- Better under high concurrency  

### Why simdjson-go instead of encoding/json?

- 3–5× faster  
- SIMD acceleration  
- Lower CPU usage  

### Why msgpack RPC instead of gRPC?

- Smaller payloads  
- Faster serialization  
- No protobuf boilerplate  
- Faster cold start  
- Simpler deployment  

### Why thin brokers instead of direct DB access?

- Centralized connection pools  
- Centralized prepared statements  
- Centralized error handling  
- Centralized metrics  
- Smaller endpoint/subscriber code  
- Easier portability  

### Why scratch containers?

- Smallest possible image  
- Fastest cold start  
- Minimal attack surface  

### Why Prometheus + Loki?

- Cloud-portable  
- Kubernetes-native  
- Works in kind/on-prem/cloud  

### Why KEDA instead of HPA?

HPA cannot scale based on Kafka lag.

### Why kind for development?

- Fast  
- Realistic  
- Kubernetes-native  
- Works with local registries  

---

## Data Flow

### Endpoint Flow

HTTP → JSON → simdjson → RPC (msgpack) → Broker → Datastore

### Subscriber Flow

Kafka → event → simdjson → RPC (msgpack) → Broker → Datastore

### Broker Flow

RPC → handler → datastore → response → msgpack → caller

---

## Metrics Overview

Endpoints, subscribers, and brokers expose:

- http_request_latency_ms  
- event_processing_latency_ms  
- sql_query_latency_ms  
- redis_command_latency_ms  
- ipc_operation_latency_ms  
- rpc_latency_ms  
- kafka_consumer_lag  
- pool_saturation_pct  

Full details in `observability.md`.

---

## Log Schema Overview

All logs use zerolog JSON:

- timestamp  
- service_name  
- http_method  
- http_path  
- rpc_target  
- rpc_latency_ms  
- sql_query_name  
- kafka_topic  
- kafka_lag  
- error_type  

Full details in `observability.md`.

---
