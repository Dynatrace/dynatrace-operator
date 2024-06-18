## Generates a manifest for Kubernetes solely for a CSI driver deployment
manifests/kubernetes/csi:
	# Generate kubernetes-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set image="$(IMAGE_URI)" > "$(KUBERNETES_CSIDRIVER_YAML)"

## Generates a Kubernetes manifest with a CRD
manifests/kubernetes/core: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
		  --namespace dynatrace \
		  --set csidriver.enabled=false \
		  --set installCRD=true \
		  --set platform="kubernetes" \
		  --set manifests=true \
		  --set olm="${OLM}" \
		  --set image="$(IMAGE_URI)" > "$(KUBERNETES_CORE_YAML)"

## Generates a manifest for Kubernetes including a CRD, a CSI driver deployment
manifests/kubernetes: manifests/kubernetes/core manifests/kubernetes/csi
	cat "$(KUBERNETES_CORE_YAML)" "$(KUBERNETES_CSIDRIVER_YAML)" > "$(KUBERNETES_ALL_YAML)"

## Generates a manifest for Kubernetes including OLM version
manifests/kubernetes/olm: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="kubernetes" \
			--set manifests=true \
			--set olm="${OLM}" \
			--set image="$(IMAGE_URI)" > "$(KUBERNETES_OLM_YAML)"

