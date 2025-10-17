#!/usr/bin/env bash

set -e

k8s_version=1.34
name=kind

# TODO: we can use renovate to update these image versions
# DAQ-15444
# based on release page https://github.com/kubernetes-sigs/kind/releases
KIND_IMAGE_K8S_131=docker.io/kindest/node:v1.31.12@sha256:0f5cc49c5e73c0c2bb6e2df56e7df189240d83cf94edfa30946482eb08ec57d2
KIND_IMAGE_K8S_132=docker.io/kindest/node:v1.32.8@sha256:abd489f042d2b644e2d033f5c2d900bc707798d075e8186cb65e3f1367a9d5a1
KIND_IMAGE_K8S_133=docker.io/kindest/node:v1.33.4@sha256:25a6018e48dfcaee478f4a59af81157a437f15e6e140bf103f85a2e7cd0cbbf2
KIND_IMAGE_K8S_134=docker.io/kindest/node:v1.34.0@sha256:7416a61b42b1662ca6ca89f02028ac133a309a2a30ba309614e8ec94d976dc5a

kind_cluster_name=${name}

if printenv K8S_VERSION >/dev/null && [ -n "$K8S_VERSION" ]; then
  k8s_version="$K8S_VERSION"
fi

case "$k8s_version" in
1.31*) image=$KIND_IMAGE_K8S_131 ;;
1.32*) image=$KIND_IMAGE_K8S_132 ;;
1.33*) image=$KIND_IMAGE_K8S_133 ;;
1.34*) image=$KIND_IMAGE_K8S_134 ;;
v*) printf "${red}${redcross}Error${end}: Kubernetes version must be given without the leading 'v'\n" >&2 && exit 1 ;;
*) printf "${red}${redcross}Error${end}: unsupported Kubernetes version ${yel}${k8s_version}${end}\n" >&2 && exit 1 ;;
esac

echo "$image"

setup_kind() {
  # (0) If kind is not installed, install it
  if ! command -v kind >/dev/null 2>&1; then
    printf "${red}${redcross}Error${end}: kind is not installed. Please install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation${end}\n" >&2
    exit 1
  fi

  # (1) Does the kind cluster already exist?
  if ! kind get clusters -q | grep -q "^$kind_cluster_name\$"; then
    kind create cluster --config "./hack/kind/cluster.yaml" \
      --image "$image" \
      --name "$kind_cluster_name"
  fi

  # (2) Does the kube config contain the context for this existing kind cluster?
  if ! kubectl config get-contexts -oname 2>/dev/null | grep -q "^kind-${kind_cluster_name}$"; then
    printf "${red}${redcross}Error${end}: the kind cluster ${yel}$kind_cluster_name${end} already exists, but your current kube config does not contain the context ${yel}kind-$kind_cluster_name${end}. Run:\n" >&2
    printf "    ${cyan}kind delete cluster --name $kind_cluster_name${end}\n" >&2
    printf "and then retry.\n"
    exit 1
  fi

  # (3) Is the existing kind cluster selected as the current context in the kube
  # config?
  if [ "$(kubectl config current-context 2>/dev/null)" != "kind-$kind_cluster_name" ]; then
    printf "${red}${redcross}Error${end}: the kind cluster ${yel}$kind_cluster_name${end} already exists, but is not selected as your current context. Run:\n" >&2
    printf "    ${cyan}kubectl config use-context kind-$kind_cluster_name${end}\n" >&2
    exit 1
  fi

  # (4) Is the current context responding?
  if ! kubectl --context "kind-$kind_cluster_name" get nodes >/dev/null 2>&1; then
    printf "${red}${redcross}Error${end}: the kind cluster $kind_cluster_name isn't responding. Please run:\n" >&2
    printf "    ${cyan}kind delete cluster --name $kind_cluster_name${end}\n" >&2
    printf "and then retry.\n"
    exit 1
  fi

  # (5) Does the current context have the correct Kubernetes version?
  existing_version=$(kubectl version -oyaml | yq e '.serverVersion | .major +"."+ .minor' -)
  if ! [[ "$k8s_version" =~ ${existing_version//./\.} ]]; then
    printf "${yel}${warn}Warning${end}: your current kind cluster runs Kubernetes %s, but %s is the expected version. Run:\n" "$existing_version" "$k8s_version" >&2
    printf "    ${cyan}kind delete cluster --name $kind_cluster_name${end}\n" >&2
    printf "and then retry.\n" >&2
  fi
}

setup_kind
