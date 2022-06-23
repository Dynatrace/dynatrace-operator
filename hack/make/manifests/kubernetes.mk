-include ../prerequisites.mk
-include ../images.mk
-include config.mk
-include manifests.mk
-include crd.mk

## Generates a manifest for Kubernetes solely for a CSI driver deployment
manifests/kubernetes/csi:
	# Generate kubernetes-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set image="$(MASTER_IMAGE)" > "$(KUBERNETES_CSIDRIVER_YAML)"

## Generates an Kubernetes manifest with a CRD
manifests/kubernetes/core: manifests/crd/helm prerequisites/kustomize
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="kubernetes" \
			--set manifests=true \
			--set olm="${OLM}" \
			--set image="$(MASTER_IMAGE)" > "$(KUBERNETES_CORE_YAML)"

## Generates a manifest for Kubernetes including a CRD, a CSI driver deployment and a OLM version
manifests/kubernetes: manifests/kubernetes/core manifests/kubernetes/csi
	cp "$(KUBERNETES_CORE_YAML)" "$(KUBERNETES_OLM_YAML)"
	cat "$(KUBERNETES_CORE_YAML)" "$(KUBERNETES_CSIDRIVER_YAML)" > "$(KUBERNETES_ALL_YAML)"


