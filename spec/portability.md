# Portability Specification

## Overview

This specification defines the portability guarantees of the system across all supported Kubernetes environments:

- kind (local development)
- on‑prem (upstream Kubernetes installed via kubeadm/kubelet/kubectl)
- cloud (managed Kubernetes: AKS, EKS, GKE)

The architecture is intentionally designed so that **workloads behave identically** across all environments.  
Differences exist only in **infrastructure**, not in **application logic**, **configuration**, or **deployment semantics**.

Portability is achieved through strict adherence to **upstream Kubernetes APIs**, **cloud‑neutral manifests**, and **environment overlays**.

---

## Portability Principles

### 1. Upstream Kubernetes Only

All manifests, controllers, and workloads rely exclusively on:

- upstream Kubernetes APIs  
- upstream CRDs (KEDA)  
- upstream ingress controllers (NGINX)  
- upstream storage interfaces (CSI)  

No vendor‑specific APIs are used. `MethodIngress` is an application-specific
extension, not a Kubernetes upstream API; its CRD is installed from
`/k8s/base/crd.yaml` and is available in each environment where the base
manifests are applied.

`MethodIngress` describes HTTP method/path/backend rules for a standard
Ingress. The controller translates those rules into ingress-nginx server
snippets; this requires the same compatible ingress controller and snippet
policy in each environment. See the
[controller documentation](../shared/methodingress-controller/README.md) for
the schema and current behavior.

This ensures:

- identical pod lifecycle  
- identical scheduling behavior  
- identical config injection  
- identical service discovery  
- identical autoscaling semantics  

---

### 2. Environment‑Neutral Base Manifests

All manifests in `/k8s/base` are **environment‑neutral**:

- Deployments  
- StatefulSets  
- Services  
- ConfigMaps  
- Secrets  
- Ingress  
- Prometheus/Loki/Grafana  
- KEDA ScaledObjects  

These manifests work **unchanged** in:

- kind  
- on‑prem  
- cloud  

Environment differences are handled exclusively through overlays.

---

### 3. Environment Overlays

Environment‑specific differences are isolated in:

- `/k8s/overlays/kind`  
- `/k8s/overlays/onprem`  
- `/k8s/overlays/cloud`  

Overlays modify only:

- storage class  
- ingress class  
- load balancer type  
- node selectors  
- autoscaling parameters  

This ensures:

- base manifests remain portable  
- overlays remain minimal  
- environment differences never leak into application code  

---

## Portability Guarantees

### 1. Identical Configuration Behavior

Configuration is delivered through:

- ConfigMaps  
- Secrets  
- env vars  
- downward API  

These behave identically across:

- kind  
- on‑prem  
- cloud  

There is **no environment‑specific configuration logic**.

See **[configuration](ca://s?q=Show_configuration_spec)** for details.

---

### 2. Identical Observability Behavior

Prometheus, Loki, Grafana, and Fluent Bit behave identically across all environments.

Differences exist only in:

- storage class  
- retention settings  
- node topology  

See **[observability](ca://s?q=Show_observability_spec)** for details.

---

### 3. Identical Autoscaling Behavior

KEDA is used as the primary autoscaler across all environments.

Differences:

| Environment | Autoscaling |
| ------------ | ------------- |
| kind | KEDA only |
| on‑prem | KEDA + HPA |
| cloud | KEDA + HPA + cloud autoscaler |

Workload behavior remains identical.

See **[keda-autoscaling](ca://s?q=Show_keda_autoscaling_doc)** for details.

---

### 4. Identical RPC Behavior

RPC between endpoints, subscribers, and brokers uses:

- msgpack  
- TCP  
- request/response pattern  
- identical error semantics  
- identical metrics/logging  

No environment‑specific networking logic is required.

See **[rpc-protocol](ca://s?q=Show_rpc_protocol_doc)** for details.

---

### 5. Identical Broker Behavior

SQL, Redis, and IPC brokers behave identically across environments.

Differences exist only in:

- storage class  
- node topology  
- persistence layer  

Broker RPC semantics remain identical.

See **[dataclient](ca://s?q=Show_dataclient_spec)** for details.

---

## Non‑Portable Components (Explicit List)

The following components are **not portable** and must be isolated in overlays:

| Component | Reason |
| ---------- | -------- |
| LoadBalancer type | cloud vs on‑prem vs kind differences |
| Storage class | environment‑specific CSI drivers |
| Node selectors | environment‑specific node pools |
| Ingress class | cloud LB vs MetalLB vs NodePort |
| Autoscaling | cloud autoscaler availability |

These components **must never appear in `/k8s/base`**.

---

## Portable Components (Explicit List)

The following components are **fully portable** and must appear only in `/k8s/base`:

| Component |
| ---------- |
| Deployments |
| StatefulSets |
| Services |
| ConfigMaps |
| Secrets |
| Ingress (generic) |
| Prometheus |
| Loki |
| Grafana |
| Fluent Bit |
| KEDA ScaledObjects |
| RBAC |
| PodDisruptionBudgets |
| HorizontalPodAutoscalers (generic) |

---

## Portability Test Matrix

| Component | kind | on‑prem | cloud | Portable? |
| ---------- | ------ | --------- | ------- | ----------- |
| Deployments | ✓ | ✓ | ✓ | Yes |
| Services | ✓ | ✓ | ✓ | Yes |
| ConfigMaps | ✓ | ✓ | ✓ | Yes |
| Secrets | ✓ | ✓ | ✓ | Yes |
| Ingress | ✓ | ✓ | ✓ | Yes |
| StorageClass | ✗ | ✗ | ✗ | No |
| LoadBalancer | ✗ | ✗ | ✗ | No |
| Node selectors | ✗ | ✗ | ✗ | No |
| Autoscaling | ✓ | ✓ | ✓ | Yes (with overlays) |
| KEDA | ✓ | ✓ | ✓ | Yes |

---

## Summary

Portability is achieved through:

- upstream Kubernetes only  
- environment‑neutral base manifests  
- minimal overlays  
- identical configuration behavior  
- identical workload behavior  
- strict separation of portable vs non‑portable components  

This ensures the entire system can run:

- locally (kind)  
- on‑prem (kubeadm)  
- in the cloud (AKS/EKS/GKE)  

with **no changes to application code** and **minimal changes to manifests**.
