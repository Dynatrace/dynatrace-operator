manifests/prepare-directory:
	find $(MANIFESTS_DIR) -type f -not -name 'kustomization.yaml' -delete

## Generates manifests e.g. CRD, RBAC etc, for Kubernetes and OpenShift
manifests: manifests/prepare-directory manifests/kubernetes manifests/openshift

## Generate manifests for the branch's image tag
manifests/branch: export MASTER_IMAGE=quay.io/dynatrace/dynatrace-operator:snapshot-${SNAPSHOT_SUFFIX}
manifests/branch: manifests
