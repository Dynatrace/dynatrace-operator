manifests/prepare-directory:
	find $(MANIFESTS_DIR) -type f -not -name 'kustomization.yaml' -delete

## Generates manifests e.g. CRD, RBAC etc, for Kubernetes and OpenShift
manifests: manifests/prepare-directory manifests/kubernetes manifests/openshift

## Generate deep copy files
manifests/deepcopy:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
