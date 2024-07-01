## Generates a manifest for Openshift solely for a CSI driver deployment
manifests/openshift/csi:
	# Generate openshift-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--show-only templates/Common/csi/clusterrole-csi.yaml \
		--show-only templates/Common/csi/csidriver.yaml \
		--show-only templates/Common/csi/daemonset.yaml \
		--show-only templates/Common/csi/priority-class.yaml \
		--show-only templates/Common/csi/role-csi.yaml \
		--show-only templates/Common/csi/serviceaccount-csi.yaml \
		--namespace dynatrace \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set image="$(IMAGE_URI)" > "$(OPENSHIFT_CSIDRIVER_YAML)"

## Generates an OpenShift manifest with a CRD
manifests/openshift/core: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
		  --namespace dynatrace \
		  --set csidriver.enabled=false \
		  --set installCRD=true \
		  --set platform="openshift" \
		  --set manifests=true \
		  --set olm="${OLM}" \
		  --set image="$(IMAGE_URI)" > "$(OPENSHIFT_CORE_YAML)"

## Generates a manifest for OpenShift including a CRD and a CSI driver deployment
manifests/openshift: manifests/openshift/core manifests/openshift/csi
	cat "$(OPENSHIFT_CORE_YAML)" "$(OPENSHIFT_CSIDRIVER_YAML)" > "$(OPENSHIFT_ALL_YAML)"

## Generates an OpenShift manifest with a CRD
manifests/openshift/olm: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set installCRD=true \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set image="$(IMAGE_URI)" > "$(OPENSHIFT_OLM_YAML)"
