# microendpoint-stack

A Go and Kubernetes foundation for independently deployable micro-endpoints,
event subscribers, and data-access brokers. The design uses small services,
MessagePack RPC, and environment-specific Kubernetes overlays to support local
development and portable deployments.

> **Implementation status:** This repository is under active development. The
> `inventory.adjust` endpoint is a request-validation scaffold and returns
> HTTP 501; it does not update inventory. SQL, Redis, and IPC broker designs
> are documented in `spec/` but their broker services are not implemented.
> Consult the component READMEs for the current behavior and limitations.

## Components

- **MethodIngress controller** — a Kubernetes controller that translates
  method/path rules into ingress-nginx server snippets.
- **Shared Go modules** — RPC transport, structured logging, and Prometheus
  metrics.
- **Kafka publishing broker** — a MessagePack-over-TCP RPC service that
  publishes messages to Kafka.
- **`inventory.adjust` endpoint** — an HTTP endpoint scaffold with request
  validation and HTTP, gRPC, and Kafka transport packages.
- **Kubernetes manifests** — base resources and `kind`, `onprem`, and `cloud`
  Kustomize overlays.

See [the architecture specification](spec/architecture.md) for the intended
system design, and [the specifications](spec/) for component contracts and
designs.

## Repository layout

```text
shared/       Reusable Go modules and MethodIngress controller
services/     Endpoint and broker services
k8s/          Kubernetes base manifests, components, and overlays
spec/         Architecture and component specifications
.github/      CI workflows and helper scripts
```

## Requirements

- Go version declared in `go.work` and the component `go.mod` files
- GNU Make
- Docker (for container builds and the kind deployment target)
- `kind` and `kubectl` (for local Kubernetes deployment)

## Build and test

Run from the repository root:

```sh
make build
make test
make vet
```

These root targets build, test, and vet the shared modules, MethodIngress
controller, and Kafka publishing broker. They do not include every service.
For the inventory endpoint, run its checks separately:

```sh
make -C services/endpoints/inventory.adjust build test vet
```

Build an individual service container with:

```sh
make -C services/brokers/kafka-publisher image
make -C services/endpoints/inventory.adjust image
```

## Kubernetes

The base manifests and environment overlays are under `k8s/`. Render an overlay
without applying it using:

```sh
kubectl kustomize k8s/overlays/kind
```

The `make deploy-kind` target builds the Kafka publisher image, creates a
temporary kind cluster, applies the kind overlay, waits for Kafka and the
publisher deployment, then deletes the cluster. It requires Docker, kind,
kubectl, and Bash.

For the endpoint-specific deployment steps and example request, see
[`services/endpoints/inventory.adjust/README.md`](services/endpoints/inventory.adjust/README.md).
The endpoint currently responds with HTTP 501 for a valid scaffold request.
For MethodIngress prerequisites and routing limitations, see
[`shared/methodingress-controller/README.md`](shared/methodingress-controller/README.md).

## Design documents

- [Architecture](spec/architecture.md)
- [Cluster environments](spec/cluster.md)
- [Configuration](spec/config.md)
- [Observability](spec/observability.md)
- [Portability](spec/portability.md)
- [Endpoint and broker specifications](spec/)
