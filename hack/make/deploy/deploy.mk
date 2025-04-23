ENABLE_CSI ?= true
PLATFORM ?= "kubernetes"

## Deploy the operator without the csi-driver
deploy/no-csi:
	@make ENABLE_CSI=false $(@D)

deploy/fips:
	@make IMAGE_URI="$(IMAGE_URI)"-fips $(@D)

## Deploy the operator with csi-driver
deploy: manifests/crd/helm
	helm upgrade dynatrace-operator config/helm/chart/default \
			--install \
			--namespace dynatrace \
			--create-namespace \
			--atomic \
			--set installCRD=true \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" \
			--set debug=$(DEBUG)

## Undeploy the current operator installation
undeploy:
	kubectl delete dynakube --all -n dynatrace
	kubectl delete edgeconnect --all -n dynatrace
	kubectl -n dynatrace wait pod --for=delete -l app.kubernetes.io/managed-by=dynatrace-operator --timeout=300s

	helm uninstall dynatrace-operator \
			--namespace dynatrace
