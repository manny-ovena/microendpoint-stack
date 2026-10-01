# MethodIngress controller

The controller adds a namespaced `MethodIngress` custom resource for describing
HTTP method and path rules associated with a standard Kubernetes Ingress.

## Container build

Run `make -C shared/methodingress-controller image` from the repository root.
The Docker build uses the repository root as its context and creates a build
workspace containing the controller and `shared/logging` modules. Local shared
modules resolve through that workspace without module `replace` directives.
Unrelated services and the host's `go.work` are not required by this image.

Install the controller-gen tool used to generate Kubernetes API artifacts:

```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
```

## Custom resource definition

The CRD is installed from [`k8s/base/crd.yaml`](../../k8s/base/crd.yaml) and is
included by the base Kustomize configuration. Its API identity is:

| Property | Value |
| --- | --- |
| API group/version | `networking.microendpoints.ovena.io/v1alpha1` |
| Kind | `MethodIngress` |
| Resource names | `methodingresses` / `methodingress` |
| Short name | `ming` |
| Scope | Namespaced |

Each resource has a `spec` with:

| Field | Type | Meaning |
| --- | --- | --- |
| `ingressRef` | string (required) | Name of a standard Ingress in the same namespace |
| `rules` | array (required) | Method-aware rules associated with that Ingress |
| `rules[].path` | string | Request path |
| `rules[].method` | string | HTTP method: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `CONNECT`, or `TRACE` |
| `rules[].backend.serviceName` | string | Backend Service name |
| `rules[].backend.servicePort` | integer | Backend Service port |

The schema requires a non-empty `ingressRef` and at least one rule. Each rule
requires `path`, `method`, `serviceName`, and a Service port between 1 and 65535.

Example:

```yaml
apiVersion: networking.microendpoints.ovena.io/v1alpha1
kind: MethodIngress
metadata:
  name: example
  namespace: default
spec:
  ingressRef: my-ingress
  rules:
    - path: /hello
      method: GET
      backend:
        serviceName: hello-service
        servicePort: 80
    - path: /hello
      method: POST
      backend:
        serviceName: hello-writer
        servicePort: 8080
```

## Routing behavior

The reconciler generates `nginx.ingress.kubernetes.io/server-snippet` on the
referenced Ingress. Users configure only `MethodIngress` rules, not NGINX text.
The Ingress and backend Services must exist in the same namespace. Use one
MethodIngress per Ingress and a dedicated hostname/server for its routes; do not
split a shared host's snippet management across multiple Ingress resources.

Each unique path gets one **exact-match** location. Within that location, the
HTTP method selects a backend, allowing GET and POST on `/hello` to route to
different Services. Unconfigured methods, including HEAD and OPTIONS, return
405; declare these methods explicitly if needed. `/hello/child` does not match
`/hello`. Existing default-backend and unrelated Ingress paths retain their
normal behavior. Do not also declare a generated path in the Ingress's own
HTTP paths: the controller rejects that collision.

The controller resolves each Service's ClusterIP and declared TCP Service port
through the Kubernetes API. This avoids DNS-resolver configuration and assumptions
about the cluster domain; users still specify Service names, never IP addresses.
Headless and ExternalName Services are not supported. The request URI, query
string, and body are preserved. Method selection uses `set` inside NGINX `if`
blocks with a single `proxy_pass` outside them.

Rules are sorted for deterministic output. Invalid rules, missing backends,
duplicate method/path pairs, and unmanaged or other-owner snippets are rejected
without overwriting the existing annotation. Ingress and Service changes trigger
reconciliation. A finalizer removes managed annotations on deletion or when
`ingressRef` changes. Reconciliation status records successfully applied routing,
not whether an ingress controller has accepted or served it.

## NGINX prerequisites

This implementation targets **ingress-nginx** and its `server-snippet`
annotation; other NGINX distributions may use different configuration APIs.
An ingress-nginx controller must be installed and manage the referenced
Ingress's class. It must allow snippet annotations, including the Critical-risk
`server-snippet`, and its annotation-value policy must permit the generated
`location`, `proxy_pass`, and `proxy_set_header` directives. Default policies
may reject them. Configure this explicitly on a trusted, dedicated ingress
installation rather than assuming annotations are enabled.

Only trusted operators should be allowed to write MethodIngress resources or
Ingress annotations. Enabling snippets permits powerful NGINX configuration;
the controller validates rule values but does not secure arbitrary annotations
written by other users. No ingress-nginx installation or policy change is made
by this module.

See [`k8s/ingress/methodingress-example.yaml`](../../k8s/ingress/methodingress-example.yaml)
for an Ingress and two colliding-path method rules. The example requires
`hello-service:80` and `hello-writer:8080` to exist first.

## Logging and verification

The controller and controller-runtime use `shared/logging` through a zerolog
logr adapter, with JSON service metadata and the standard logging environment
variables. Build within the repository's Go workspace or the Docker workspace;
the unpublished shared module does not require a `replace` directive.

Run `make -C shared/methodingress-controller test vet` for reconciliation and
generation checks. To verify the generated snippet in a real NGINX container,
including same-path GET/POST routing, 405 responses, and query preservation:

```sh
cd shared/methodingress-controller
NGINX_INTEGRATION=1 go test ./controllers -run TestSnippetInNGINX -v
```

The integration test requires Docker, uses `nginx:stable`, and removes its
temporary container afterwards. It does not install ingress-nginx in kind.