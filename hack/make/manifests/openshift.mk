-include ../prerequisites.mk
-include ../images.mk
-include config.mk
-include crd.mk

## Generates a manifest for Openshift solely for a CSI driver deployment
manifests/openshift/csi:
	# Generate openshift-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > "$(OPENSHIFT_CSIDRIVER_YAML)"

## Generates an OpenShift manifest with a CRD
manifests/openshift/core: manifests/crd/helm prerequisites/kustomize
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set installCRD=true \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > "$(OPENSHIFT_CORE_YAML)"

## Generates a manifest for OpenShift including a CRD and a CSI driver deployment
manifests/openshift: manifests/openshift/core manifests/openshift/csi
	cp "$(OPENSHIFT_CORE_YAML)" "$(OPENSHIFT_OLM_YAML)"
	cat "$(OPENSHIFT_CORE_YAML)" "$(OPENSHIFT_CSIDRIVER_YAML)" > "$(OPENSHIFT_ALL_YAML)"