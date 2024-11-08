ENABLE_CSI ?= true
PLATFORM ?= "kubernetes"

BRANCH ?= $(shell git branch --show-current)
SNAPSHOT_SUFFIX ?= $(shell echo "${BRANCH}" | sed "s/[^a-zA-Z0-9_-]/-/g")

## Deploy the operator without the csi-driver, with platform specified in % (kubernetes or openshift)
deploy/%/no-csi:
	@make ENABLE_CSI=false $(@D)

## Deploy the operator with csi-driver, with platform specified in % (kubernetes or openshift)
deploy/%:
	@make PLATFORM=$(@F) $(@D)

## Deploy the operator with csi-driver, on kubernetes
deploy: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform=$(PLATFORM) \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl apply -f -

preview/%:
	@make PLATFORM=$(@F) $(@D)

preview: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform=$(PLATFORM) \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" > operator-preview-$(PLATFORM)-$(SNAPSHOT_SUFFIX).yaml
