ENABLE_CSI ?= true
DEBUG_LOGS ?= true
HA_WEBHOOK ?= true
PLATFORM ?= "kubernetes"
HELM_CHART ?= config/helm/chart/default

## Display the image name used to deploy the helm chart
deploy/show-image-ref:
	@echo $(IMAGE_URI)

## Display the image name used to deploy the FIPS helm chart
deploy/show-image-ref/fips:
	@# Don't call make here to omit the make[1] lines for normal CLI usage.
	@echo $(IMAGE_URI)-fips

## Deploy the operator without the csi-driver
deploy/no-csi:
	@make ENABLE_CSI=false $(@D)

deploy/fips:
	@make IMAGE_URI="$(IMAGE_URI)"-fips $(@D)

## Deploy the operator with csi-driver
deploy: manifests/crd/helm
	helm upgrade dynatrace-operator $(HELM_CHART) \
			--install \
			--namespace dynatrace \
			--create-namespace \
			--atomic \
			--set installCRD=true \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set webhook.highAvailability=$(HA_WEBHOOK) \
			--set manifests=true \
			--set image=$(IMAGE_URI) \
			--set debugLogs=$(DEBUG_LOGS) \
			--set debug=$(DEBUG)

## Undeploy the current operator installation
undeploy:
	kubectl delete dynakube --all -n dynatrace || true
	kubectl delete edgeconnect --all -n dynatrace || true
	kubectl -n dynatrace wait pod --for=delete -l app.kubernetes.io/managed-by=dynatrace-operator --timeout=300s

	helm uninstall dynatrace-operator \
			--namespace dynatrace
