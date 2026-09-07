#!/usr/bin/env bash

set -e

k8s_version=1.37
name=kind

# Color codes and symbols
red="\033[0;31m"
yel="\033[0;33m"
cyan="\033[0;36m" # C = cyan
end="\033[0m" # E is the "end" marker.
warn="⚠️  "
redcross="❌  "

# TODO: we can use renovate to update these image versions
# DAQ-15444
# based on release page https://github.com/kubernetes-sigs/kind/releases
KIND_IMAGE_K8S_129=docker.io/kindest/node:v1.29.14@sha256:8703bd94ee24e51b778d5556ae310c6c0fa67d761fae6379c8e0bb480e6fea29
KIND_IMAGE_K8S_130=docker.io/kindest/node:v1.30.13@sha256:397209b3d947d154f6641f2d0ce8d473732bd91c87d9575ade99049aa33cd648
KIND_IMAGE_K8S_131=docker.io/kindest/node:v1.31.14@sha256:6f86cf509dbb42767b6e79debc3f2c32e4ee01386f0489b3b2be24b0a55aac2b
KIND_IMAGE_K8S_132=docker.io/kindest/node:v1.32.11@sha256:5fc52d52a7b9574015299724bd68f183702956aa4a2116ae75a63cb574b35af8
KIND_IMAGE_K8S_133=docker.io/kindest/node:v1.33.12@sha256:3f5c8443c620245e4d355cfe09e96a91ead32ceaa569d3f1ca9edf0cb2fe2ff4
KIND_IMAGE_K8S_134=docker.io/kindest/node:v1.34.11@sha256:44e222ee2132dab25ff87301682f89eb82c7880ea3a1bf543bfe9708fd08d67d
KIND_IMAGE_K8S_135=docker.io/kindest/node:v1.35.8@sha256:07b2536e30b803ed61d1677a79df6115f798ce64c80f9e22f6ed45afd09323c0
KIND_IMAGE_K8S_136=docker.io/kindest/node:v1.36.4@sha256:099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed
KIND_IMAGE_K8S_137=docker.io/kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5

kind_cluster_name=${name}

if printenv K8S_VERSION >/dev/null && [ -n "$K8S_VERSION" ]; then
  k8s_version="$K8S_VERSION"
fi

case "$k8s_version" in
1.29*) image=$KIND_IMAGE_K8S_129 ;;
1.30*) image=$KIND_IMAGE_K8S_130 ;;
1.31*) image=$KIND_IMAGE_K8S_131 ;;
1.32*) image=$KIND_IMAGE_K8S_132 ;;
1.33*) image=$KIND_IMAGE_K8S_133 ;;
1.34*) image=$KIND_IMAGE_K8S_134 ;;
1.35*) image=$KIND_IMAGE_K8S_135 ;;
1.36*) image=$KIND_IMAGE_K8S_136 ;;
1.37*) image=$KIND_IMAGE_K8S_137 ;;
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
