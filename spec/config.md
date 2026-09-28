# Configuration Specification

## Overview

This specification defines how configuration is supplied to all services in the system — endpoints, subscribers, and brokers — using **standard Kubernetes primitives**.

Configuration is intentionally designed to be:

- portable  
- environment‑agnostic  
- cloud‑neutral  
- compatible with kind, on‑prem Kubernetes, and cloud Kubernetes  
- simple and explicit  

Because all environments use **upstream Kubernetes**, configuration behaves **identically** across:

- kind (local development)
- on‑prem Kubernetes (native kubeadm‑based clusters)
- cloud Kubernetes (AKS, EKS, GKE)

No environment‑specific configuration logic is required.

---

## Configuration Sources

All configuration is delivered through the following Kubernetes primitives:

### 1. Environment Variables

Used for:

- connection strings  
- feature flags  
- RPC timeouts  
- broker endpoints  
- service metadata  

### 2. ConfigMaps

Used for:

- non‑secret configuration  
- service defaults  
- tuning parameters  
- environment profiles  

### 3. Secrets

Used for:

- database passwords  
- Redis auth  
- Kafka SASL credentials (if applicable)  
- API keys  

### 4. Downward API

Used for:

- pod name  
- namespace  
- node name  
- resource limits  

### 5. Volume Mounts (optional)

Used only when:

- large config files are needed  
- certificates must be mounted  
- IPC broker paths require explicit mounts  

---

## Environment Parity

All environments — kind, on‑prem, cloud — support the same configuration mechanisms.

| Environment | Description | Config Behavior |
| ------------ | ------------- | ----------------- |
| kind | Local Kubernetes cluster for development | identical |
| on‑prem | Native kubeadm‑based Kubernetes cluster | identical |
| cloud | Managed Kubernetes (AKS/EKS/GKE) | identical |

There are **no differences** in:

- how ConfigMaps are mounted  
- how Secrets are injected  
- how env vars are passed  
- how downward API works  
- how volumes are mounted  

This is a core portability guarantee of the architecture.

---

## Environment‑Specific Values (Not Mechanisms)

While configuration **mechanisms** are identical, configuration **values** may differ.

### 1. Service Endpoints

- kind: `postgres.kind.svc.cluster.local`
- on‑prem: `postgres.onprem.svc.cluster.local`
- cloud: `postgres.cloud.svc.cluster.local`

### 2. Storage Classes

- kind: `local-path`
- on‑prem: `nfs`, `ceph-rbd`, `longhorn`
- cloud: `gp2`, `pd-standard`, `azure-disk`

### 3. Ingress Class

- kind: `nginx`
- on‑prem: `nginx` or `haproxy`
- cloud: `nginx` or cloud LB

### 4. Autoscaling

- kind: KEDA only  
- on‑prem: KEDA + HPA  
- cloud: KEDA + HPA + cloud autoscaler  

These differences are **values**, not **mechanisms**.

---

## Configuration Precedence

Configuration follows a strict precedence order:

1. **Environment Variables**  
2. **Secrets**  
3. **ConfigMaps**  
4. **Default Values (in code)**  

This ensures:

- secure values override non‑secure values  
- environment‑specific overrides defaults  
- no ambiguity in runtime behavior  

---

## Required Configuration Fields

### Common Fields (All Services)

| Field | Description |
| ------- | ------------- |
| SERVICE_NAME | Logical service name |
| SERVICE_ENV | dev / kind / onprem / cloud |
| LOG_LEVEL | debug / info / warn / error |
| RPC_TIMEOUT_MS | RPC timeout |
| RPC_MAX_RETRIES | Retry count |

### Endpoint Fields

| Field | Description |
| ------- | ------------- |
| HTTP_PORT | Port to bind |
| BROKER_TARGET | sql / redis / ipc |
| MAX_REQUEST_SIZE_BYTES | Request size limit |

### Subscriber Fields

| Field | Description |
| ------- | ------------- |
| KAFKA_TOPIC | Kafka topic |
| KAFKA_GROUP | Consumer group |
| KAFKA_BROKERS | Broker list |
| MAX_EVENT_SIZE_BYTES | Event size limit |

### Broker Fields

| Field | Description |
| ------- | ------------- |
| SQL_DSN | Postgres DSN |
| REDIS_ADDR | Redis address |
| IPC_BASE_PATH | Filesystem base path |

---

## Validation Rules

### Required

- SERVICE_NAME  
- SERVICE_ENV  
- RPC_TIMEOUT_MS  
- BROKER_TARGET (endpoints)  
- KAFKA_TOPIC (subscribers)  
- SQL_DSN / REDIS_ADDR / IPC_BASE_PATH (brokers)  

### Optional

- LOG_LEVEL  
- MAX_REQUEST_SIZE_BYTES  
- MAX_EVENT_SIZE_BYTES  

### Defaulting Behavior

- LOG_LEVEL defaults to `info`  
- RPC_TIMEOUT_MS defaults to `250`  
- RPC_MAX_RETRIES defaults to `3`  

---

## Configuration Examples

### Endpoint Example (kind)

```yaml
env:
  - name: SERVICE_ENV
    value: "kind"
  - name: BROKER_TARGET
    value: "sql"
  - name: SQL_DSN
    valueFrom:
      secretKeyRef:
        name: postgres-secret
        key: dsn
```

### Subscriber Example (on‑prem)

```yaml
env:
  - name: SERVICE_ENV
    value: "onprem"
  - name: KAFKA_TOPIC
    value: "order.placed"
  - name: KAFKA_BROKERS
    value: "kafka.onprem.svc.cluster.local:9092"
```

### Broker Example (cloud)

```yaml
env:
  - name: SERVICE_ENV
    value: "cloud"
  - name: SQL_DSN
    valueFrom:
      secretKeyRef:
        name: cloud-postgres-secret
        key: dsn
```
