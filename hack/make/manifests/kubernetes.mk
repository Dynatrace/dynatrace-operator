define generate_k8s_manifest
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set csidriver.enabled=$(1) \
		--set installCRD=true \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm=$(OLM) \
		--set image=$(IMAGE_URI) > $(2)
endef

## Generates a Kubernetes manifest including CRD and CSI driver
manifests/kubernetes/csi: manifests/crd/helm
	$(call generate_k8s_manifest,true,$(KUBERNETES_CSIDRIVER_YAML))

## Generates a Kubernetes manifest including CRD without CSI driver
manifests/kubernetes/core: manifests/crd/helm
	$(call generate_k8s_manifest,false,$(KUBERNETES_CORE_YAML))

## Generates a Kubernetes manifest including CRD and CSI driver with OLM set to true
manifests/kubernetes/olm: manifests/crd/helm
	OLM=true $(call generate_k8s_manifest,true,$(KUBERNETES_OLM_YAML))

## Generates a manifest for Kubernetes including a CRD, a CSI driver deployment
manifests/kubernetes: manifests/kubernetes/core manifests/kubernetes/csi
