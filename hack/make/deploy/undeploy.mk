## Remove the operator without the csi-driver, with platform specified in % (kubernetes or openshift)
undeploy/%/no-csi:
	@make ENABLE_CSI=false $(@D)

## Remove the operator with csi-driver, with platform specified in % (kubernetes or openshift)
undeploy/%:
	@make PLATFORM=$(@F) $(@D)

## Remove the operator with csi-driver, on kubernetes
undeploy: manifests/crd/helm
	kubectl delete dynakube --all -n dynatrace
	kubectl -n dynatrace wait pod --for=delete -l app.kubernetes.io/managed-by=dynatrace-operator --timeout=300s
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform=$(PLATFORM) \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl delete -f -
