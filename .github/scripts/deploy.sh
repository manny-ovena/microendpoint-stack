#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cluster_name="ci-methodingress-${GITHUB_RUN_ID:-$$}-${GITHUB_RUN_ATTEMPT:-1}"
publisher_image="ghcr.io/manny-ovena/kafka-publisher:local"

kubectl kustomize "$repo_root/k8s/overlays/kind" >/dev/null
make -C "$repo_root/services/brokers/kafka-publisher" image IMAGE="$publisher_image"
kind create cluster --name "$cluster_name" --wait 120s
trap 'kind delete cluster --name "$cluster_name"' EXIT

kind load docker-image "$publisher_image" --name "$cluster_name"
kubectl --context "kind-$cluster_name" apply -k "$repo_root/k8s/overlays/kind"
kubectl --context "kind-$cluster_name" -n database rollout status statefulset/kafka --timeout=240s
kubectl --context "kind-$cluster_name" -n app rollout status deployment/kafka-publisher --timeout=180s
