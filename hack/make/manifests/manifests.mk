-include config.mk
-include kubernetes.mk
-include openshift.mk

manifests/prepare-directory:
	find $(MANIFESTS_DIR) -type f -not -name 'kustomization.yaml' -delete

# Generates manifests e.g. CRD, RBAC etc, for Kubernetes and OpenShift
manifests: manifests/prepare-directory manifests/kubernetes manifests/openshift
