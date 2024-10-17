
DEBUG ?= false

## Deploy the operator in a cluster configured in ~/.kube/config where platform and version are autodetected
deploy/helm: manifests/crd/helm
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

## Undeploy the operator in a cluster configured in ~/.kube/config where platform and k8s version are autodetected
undeploy/helm:
	helm uninstall dynatrace-operator \
			--namespace dynatrace
