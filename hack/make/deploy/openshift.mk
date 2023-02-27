## Deploy the operator in the OpenShift cluster configured in ~/.kube/config
deploy/openshift: manifests/crd/helm
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set csidriver.enabled=true \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | oc apply -f -

deploy/openshift-no-csi: manifests/crd/helm
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | oc apply -f -


undeploy/openshift: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set csidriver.enabled=true \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | oc delete -f -

undeploy/openshift-no-csi: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="openshift" \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | oc delete -f -
