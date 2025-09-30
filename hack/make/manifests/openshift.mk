define generate_openshift_manifest
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set csidriver.enabled=$(1) \
		--set installCRD=true \
		--set platform="openshift" \
		--set manifests=true \
		--set olm=${OLM} \
		--set image="$(IMAGE_URI)" > $(2)
endef

## Generates an Openshift manifest including CRD and CSI driver
manifests/openshift/csi: manifests/crd/helm
	$(call generate_openshift_manifest,true,$(OPENSHIFT_CSIDRIVER_YAML))

## Generates a Openshift manifest including CRD without CSI driver
manifests/openshift/core: manifests/crd/helm
	$(call generate_openshift_manifest,false,$(OPENSHIFT_CORE_YAML))

## Generates an Openshift manifest including CRD and CSI driver with OLM set to true
manifests/openshift/olm: manifests/crd/helm
	OLM=true $(call generate_openshift_manifest,true,$(OPENSHIFT_OLM_YAML))

## Generates a manifest for OpenShift including a CRD and a CSI driver deployment
manifests/openshift: manifests/openshift/core manifests/openshift/csi
