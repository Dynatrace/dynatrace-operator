## Deploy the operator in the OpenShift cluster configured in ~/.kube/config
deploy/openshift: manifests/crd/helm
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set csidriver.enabled=true \
			--set manifests=true \
			--set image="$(BRANCH_IMAGE)" | oc apply -f -
