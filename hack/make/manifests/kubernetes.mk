-include ../prerequisites.mk
-include ../images.mk
-include config.mk
-include crd.mk

## Generates a manifest for Kubernetes solely for a CSI driver deployment
manifests/kubernetes/csi:
	# Generate kubernetes-csidriver.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set operator.image="$(MASTER_IMAGE)" > "$(KUBERNETES_CSIDRIVER_YAML)"

	grep -v 'app.kubernetes.io/managed-by' "$(KUBERNETES_CSIDRIVER_YAML)"  > config/deploy/kubernetes/tmp.yaml
	grep -v 'helm.sh' config/deploy/kubernetes/tmp.yaml > "$(KUBERNETES_CSIDRIVER_YAML)"
	rm config/deploy/kubernetes/tmp.yaml

## Generates a manifest for Kubernetes including a CRD and a CSI driver deployment
manifests/kubernetes: manifests/kubernetes/crd manifests/kubernetes/csi
	cp "$(KUBERNETES_CRD_AND_OTHERS_YAML)" "$(KUBERNETES_OLM_YAML)"
	cat "$(KUBERNETES_CRD_AND_OTHERS_YAML)" "$(KUBERNETES_CSIDRIVER_YAML)" > "$(KUBERNETES_ALL_YAML)"
