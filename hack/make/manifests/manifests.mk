manifests/prepare-directory:
	find $(MANIFESTS_DIR) -type f -not -name 'kustomization.yaml' -delete

## Generates manifests e.g. CRD, RBAC etc, for Kubernetes and OpenShift
manifests: manifests/prepare-directory manifests/kubernetes manifests/openshift manifests/deepcopy

## Generate deep copy files
manifests/deepcopy: prerequisites/controller-gen
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./pkg/api/..."
