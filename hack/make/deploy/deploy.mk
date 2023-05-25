ENABLE_CSI ?= true
PLATFORM ?= "kubernetes"

deploy/%/no-csi:
	@make ENABLE_CSI=false $(@D)

deploy/%:
	@make PLATFORM=$(@F) $(@D)

deploy: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform=$(PLATFORM) \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl apply -f -
