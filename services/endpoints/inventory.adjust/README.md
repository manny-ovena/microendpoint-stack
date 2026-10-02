# inventory.adjust

HTTP endpoint for adjusting a SKU's inventory. See
[`spec/endpoints/inventory.adjust.md`](../../../spec/endpoints/inventory.adjust.md).

This scaffold validates requests and exposes `POST /inventory/adjust`; the
inventory update is not implemented until the SQL broker is connected.

## Container

Run `make image` in this directory to build the non-root scratch image.
The build uses the repository root as its context to include `shared/logging`.
Override the tag with `make image IMAGE=your-image:tag`.

## kind

From the repository root, build and load the image into your cluster:

```sh
make -C services/endpoints/inventory.adjust image
kind load docker-image ghcr.io/manny-ovena/inventory-adjust:local --name microendpoint-stack
kind export kubeconfig --name microendpoint-stack
kubectl create namespace app --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -k k8s/endpoints/inventory.adjust
kubectl -n app set image deployment/inventory-adjust inventory-adjust=ghcr.io/manny-ovena/inventory-adjust:local
kubectl -n app rollout status deployment/inventory-adjust
kubectl -n app port-forward service/inventory-adjust 18080:80
```

Use a kubeconfig/context for the target cluster with all `kubectl` commands.
The ConfigMap provides `PORT` via `envFrom`; its value must match the declared
container port. The Service exposes port 80. ConfigMap changes require a
Deployment restart to refresh environment variables.

In another terminal, a valid scaffold request returns HTTP 501 (not a completed
inventory update):

```sh
curl -i -H 'Content-Type: application/json' \
  -d '{"sku":"test-sku","delta":1}' http://localhost:18080/inventory/adjust
```
