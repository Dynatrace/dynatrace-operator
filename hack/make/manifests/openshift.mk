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
		--set autoCreateSecret=false \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > "$(OPENSHIFT_CSIDRIVER_YAML)"

	grep -v 'app.kubernetes.io/managed-by' "$(OPENSHIFT_CSIDRIVER_YAML)"  > config/deploy/kubernetes/tmp.yaml
	grep -v 'helm.sh' config/deploy/kubernetes/tmp.yaml > "$(OPENSHIFT_CSIDRIVER_YAML)"
	rm config/deploy/kubernetes/tmp.yaml

## Generates a manifest for OpenShift including a CRD and a CSI driver deployment
manifests/openshift: manifests/openshift/crd manifests/openshift/csi
	cp "$(OPENSHIFT_CRD_AND_OTHERS_YAML)" "$(OPENSHIFT_OLM_YAML)"
	cat "$(OPENSHIFT_CRD_AND_OTHERS_YAML)" "$(OPENSHIFT_CSIDRIVER_YAML)" > "$(OPENSHIFT_ALL_YAML)"