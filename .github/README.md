# CI workflows

The executable build, test, and deploy-kind workflows live in
`.github/workflows/`. Dispatch them manually from
the Actions tab or with `gh workflow run build.yaml`, `gh workflow run test.yaml`,
or `gh workflow run deploy-kind.yaml`.

The root `Makefile` delegates build, test, and vet targets to the controller's
`Makefile`. The committed `go.work` includes the controller and shared logging
modules; add new modules to the workspace as they are created.
`make build` writes `shared/methodingress-controller/bin/manager` and builds a
local `methodingress-controller:ci` Docker image. `make test` runs `go test`
and `go vet` for the controller module. The deploy-kind workflow creates and
deletes its own kind cluster, applies the full kind overlay, and confirms
Kubernetes accepts the manifests. It does **not** wait for pods to become Ready:
the controller manifest pulls a GHCR `:latest` image, and successful apply
does not imply that image is available or that the workloads are healthy.

Run the same checks locally from the repository root with `make build`,
`make test`, and `make deploy-kind`. `make image` and `make vet` can be run
independently. The `.github/scripts/build.sh` and `.github/scripts/test.sh` scripts
remain as wrappers for callers that use them from another directory.
Building needs Go and Docker; deploying needs kind, kubectl, and Docker.

Publishing an image is a separate manual operation. After authenticating Docker
to GHCR with permission to publish the package, run from
`shared/methodingress-controller`:

```bash
docker build -t ghcr.io/manny-ovena/methodingress-controller:latest -f Dockerfile ../..
docker push ghcr.io/manny-ovena/methodingress-controller:latest
```
