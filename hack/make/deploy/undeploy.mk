undeploy/%/no-csi:
	@make ENABLE_CSI=false $(@D)

undeploy/%:
	@make PLATFORM=$(@F) $(@D)

undeploy: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform=$(PLATFORM) \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl delete -f -
