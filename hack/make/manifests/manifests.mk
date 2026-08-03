MANIFEST_NAMESPACE ?= dynatrace

kubernetes = $(KUBERNETES_CORE_YAML)
kubernetes/csi  = $(KUBERNETES_CSIDRIVER_YAML)
openshift  = $(OPENSHIFT_CORE_YAML)
openshift/csi   = $(OPENSHIFT_CSIDRIVER_YAML)

manifests/prepare-directory:
	find $(MANIFESTS_DIR) -type f -not -name 'kustomization.yaml' -delete

## Generates manifests e.g. CRD, RBAC etc, for Kubernetes and OpenShift
manifests: manifests/prepare-directory manifests/kubernetes manifests/openshift manifests/deepcopy

## Generate deep copy files
manifests/deepcopy: prerequisites/controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/api/..."

manifests/apply/%: manifests/prepare-directory manifests/kubernetes manifests/openshift
	kubectl get namespace $(MANIFEST_NAMESPACE) >/dev/null 2>&1 || kubectl create namespace $(MANIFEST_NAMESPACE)
	kubectl apply -f $($*) # Apply how a user would based on release notes

manifests/delete/%: manifests/prepare-directory manifests/kubernetes manifests/openshift
	kubectl delete --ignore-not-found -f $($*)

