## Deploy the operator in the OpenShift cluster configured in ~/.kube/config
deploy/openshift: manifests/crd/helm
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | oc apply -f -

## Deploy the operator without CSI in the OpenShift cluster configured in ~/.kube/config
deploy/openshift-no-csi:
	ENABLE_CSI=false $(MAKE) deploy/openshift

## Undeploy the operator in the OpenShift cluster configured in ~/.kube/config
undeploy/openshift: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | oc delete -f -

## Undeploy the operator without CSI in the OpenShift cluster configured in ~/.kube/config
undeploy/openshift-no-csi:
	ENABLE_CSI=false $(MAKE) undeploy/openshift
