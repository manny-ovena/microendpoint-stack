# Cluster Specification

## Overview

This specification defines the Kubernetes cluster architecture for all supported environments:

- **kind** — local development cluster  
- **on‑prem** — upstream Kubernetes installed using native tools (kubeadm, kubelet, kubectl, containerd/CRI‑O)  
- **cloud** — managed Kubernetes (AKS, EKS, GKE)

All environments use **upstream Kubernetes**, so the cluster architecture is intentionally:

- portable  
- cloud‑neutral  
- spec‑driven  
- identical in workload behavior  

Differences between environments are limited to **infrastructure**, not **application architecture**.

---

## Cluster Components

### MethodIngress Custom Resource

The base Kubernetes resources install the namespaced
`MethodIngress` CRD (`networking.microendpoints.ovena.io/v1alpha1`) and the
controller's RBAC and deployment. It associates HTTP method/path rules and
backend Service references with a standard Kubernetes Ingress. Each
`MethodIngress` refers to an Ingress in its own namespace.

The controller generates method-aware ingress-nginx server snippets on the
referenced Ingress. The ingress controller must be installed and explicitly
configured to permit these snippets.
See [`shared/methodingress-controller`](../shared/methodingress-controller/README.md)
for the resource schema, example, and implementation status.

### Control Plane

All environments include:

- API server  
- scheduler  
- controller manager  
- etcd (managed differently per environment)

#### Environment Differences

| Environment | Control Plane Characteristics |
| ------------ | -------------------------------- |
| kind | Single‑node control plane running inside Docker |
| on‑prem | Multi‑node control plane via kubeadm; etcd external or stacked |
| cloud | Managed control plane; no direct access to etcd |

---

### Worker Nodes

Worker nodes run:

- endpoints  
- subscribers  
- brokers  
- observability stack  

#### Environment Differences

| Environment | Worker Node Characteristics |
| ------------ | ------------------------------ |
| kind | Docker containers acting as nodes |
| on‑prem | Bare‑metal or VM nodes; containerd/CRI‑O |
| cloud | Cloud VMs; autoscaling groups; managed node pools |

Node selectors and taints may be used for:

- broker isolation  
- high‑throughput subscribers  
- dedicated storage nodes  

---

## Networking

All environments use:

- Kubernetes Services  
- ClusterIP for internal traffic  
- NGINX ingress controller (portable)  
- CNI plugin (Calico or Cilium recommended)

### Environment Differences

| Environment | Networking Notes |
| ------------ | ------------------ |
| kind | NodePort ingress; no cloud LB |
| on‑prem | MetalLB or HAProxy for LoadBalancer |
| cloud | Native cloud load balancers |

---

## Storage

Storage is environment‑specific but follows the same PVC/PV model.

### Storage Classes

| Environment | Storage Class |
| ------------ | ---------------- |
| kind | `local-path` |
| on‑prem | `nfs`, `ceph-rbd`, `longhorn` |
| cloud | `gp2`, `pd-standard`, `azure-disk` |

---

## Observability Stack

All environments deploy:

- Prometheus  
- Loki  
- Grafana  
- Fluent Bit (DaemonSet)

### Environment Differences

| Environment | Observability Notes |
| ------------ | ---------------------- |
| kind | Local storage; single‑node Loki |
| on‑prem | Ceph/NFS/Longhorn backing |
| cloud | Cloud storage classes; optional cloud logging integrations |

---

## Autoscaling

Autoscaling is consistent across environments, with differences in available mechanisms.

### Autoscaling Components

- KEDA (primary autoscaler for subscribers)  
- HPA (optional for endpoints)  

### Environment Differences

| Environment | Autoscaling Notes |
| ------------ | ------------------- |
| kind | KEDA only; no cloud autoscaler |
| on‑prem | KEDA + HPA |
| cloud | KEDA + HPA + cloud autoscaler |

---

## Cluster Profiles

### kind Profile

- 1 control plane  
- 2 worker nodes  
- local registry  
- local-path storage  
- NGINX ingress  
- Prometheus + Loki + Grafana  
- KEDA  

Used for:

- local development  
- reproducible testing  
- CI environments  

---

## on‑prem Profile

Upstream Kubernetes installed using:

- kubeadm  
- kubelet  
- kubectl  
- containerd or CRI‑O  

Characteristics:

- multi‑node control plane  
- bare‑metal or VM workers  
- MetalLB or HAProxy for LoadBalancer  
- Ceph/NFS/Longhorn storage  
- KEDA + HPA  

Used for:

- enterprise clusters  
- datacenter deployments  
- air‑gapped environments  

---

## cloud Profile

Managed Kubernetes:

- AKS  
- EKS  
- GKE  

Characteristics:

- managed control plane  
- managed node pools  
- cloud load balancers  
- cloud storage classes  
- KEDA + HPA + cloud autoscaler  

Used for:

- production workloads  
- global deployments  
- autoscaling environments  

---

## Cluster Portability

All manifests in `/k8s/base` are **environment‑neutral**.

Environment‑specific differences are handled via overlays:

- `/k8s/overlays/kind`  
- `/k8s/overlays/onprem`  
- `/k8s/overlays/cloud`  

This ensures:

- identical workloads  
- identical configuration mechanisms  
- identical observability  
- identical autoscaling logic  

Only infrastructure changes — not application behavior.

---
