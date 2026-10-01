#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cluster_name="ci-methodingress-${GITHUB_RUN_ID:-$$}-${GITHUB_RUN_ATTEMPT:-1}"

kubectl kustomize "$repo_root/k8s/overlays/kind" >/dev/null
kind create cluster --name "$cluster_name" --wait 120s
trap 'kind delete cluster --name "$cluster_name"' EXIT

kubectl --context "kind-$cluster_name" apply -k "$repo_root/k8s/overlays/kind"
