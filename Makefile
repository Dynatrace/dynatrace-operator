SHELL = bash

OLM = false

# Current Operator version
VERSION ?= 0.0.1
# Default platform for bundles
PLATFORM ?= openshift
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)

MASTER_IMAGE = quay.io/dynatrace/dynatrace-operator:snapshot
BRANCH_IMAGE = quay.io/dynatrace/dynatrace-operator:snapshot-$(shell git branch --show-current | sed "s/[^a-zA-Z0-9_-]/-/g")
OLM_IMAGE ?= registry.connect.redhat.com/dynatrace/dynatrace-operator:v${VERSION}

OUT ?= all

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
ifeq ($(origin IMG),undefined)
	ifneq ($(shell git branch --show-current | grep "^release-"),)
	MASTER_IMAGE=$(BRANCH_IMAGE)
	else ifeq ($(shell git branch --show-current), master)
	BRANCH_IMAGE=$(MASTER_IMAGE)
	else
	IMG=$(BRANCH_IMAGE)
	endif
endif

CRD_OPTIONS ?= "crd:preserveUnknownFields=false, crdVersions=v1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

helm-test:
	cd config/helm && ./testing/test.sh

helm-lint:
	cd config/helm && ./testing/lint.sh

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager ./src/cmd/operator/

manager-amd64: export GOOS=linux
manager-amd64: export GOARCH=amd64
manager-amd64: generate fmt vet
	go build -o bin/manager-amd64 ./src/cmd/operator/

# Run against the configured Kubernetes cluster in ~/.kube/config
run: export RUN_LOCAL=true
run: export POD_NAMESPACE=dynatrace
run: generate fmt vet manifests
	go run ./src/cmd/operator/

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	cd config/deploy/kubernetes && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build config/deploy/kubernetes | kubectl apply -f -
	make reset-kustomization-files

# Deploy controller in the configured OpenShift cluster in ~/.kube/config
deploy-ocp: manifests kustomize
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	cd config/deploy/openshift && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build config/deploy/openshift | oc apply -f -
	make reset-kustomization-files

deploy-local:
	./build/deploy_local.sh

deploy-local-easy: export TAG=snapshot-$(shell git branch --show-current | sed "s/[^a-zA-Z0-9_-]/-/g")
deploy-local-easy:
	./build/deploy_local.sh

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen kustomize
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

	# Create directories for manifests if they do not exist
	mkdir -p config/deploy/kubernetes
	mkdir -p config/deploy/openshift

	# Generate kubernetes.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set operator.image="$(MASTER_IMAGE)" > config/deploy/kubernetes/kubernetes.yaml
	grep -v 'app.kubernetes.io' config/deploy/kubernetes/kubernetes.yaml > config/deploy/kubernetes/tmp.yaml
	grep -v 'helm.sh' config/deploy/kubernetes/tmp.yaml > config/deploy/kubernetes/kubernetes.yaml
	rm config/deploy/kubernetes/tmp.yaml

	# Generate kubernetes-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set operator.image="$(MASTER_IMAGE)" > config/deploy/kubernetes/kubernetes-csi.yaml
	grep -v 'app.kubernetes.io' config/deploy/kubernetes/kubernetes-csi.yaml > config/deploy/kubernetes/tmp.yaml
	grep -v 'helm.sh' config/deploy/kubernetes/tmp.yaml > config/deploy/kubernetes/kubernetes-csi.yaml
	rm config/deploy/kubernetes/tmp.yaml

	# Generate openshift.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > config/deploy/openshift/openshift.yaml
	grep -v 'app.kubernetes.io' config/deploy/openshift/openshift.yaml > config/deploy/openshift/tmp.yaml
	grep -v 'helm.sh' config/deploy/openshift/tmp.yaml > config/deploy/openshift/openshift.yaml
	rm config/deploy/openshift/tmp.yaml

	# Generate openshift-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > config/deploy/openshift/openshift-csi.yaml
	grep -v 'app.kubernetes.io' config/deploy/openshift/openshift-csi.yaml > config/deploy/openshift/tmp.yaml
	grep -v 'helm.sh' config/deploy/openshift/tmp.yaml > config/deploy/openshift/openshift-csi.yaml
	rm config/deploy/openshift/tmp.yaml

	$(KUSTOMIZE) build config/crd | cat - config/deploy/kubernetes/kubernetes.yaml > temp
	mv temp config/deploy/kubernetes/kubernetes.yaml

	$(KUSTOMIZE) build config/crd | cat - config/deploy/openshift/openshift.yaml > temp
	mv temp config/deploy/openshift/openshift.yaml

	cat config/deploy/kubernetes/kubernetes.yaml config/deploy/kubernetes/kubernetes-csi.yaml > config/deploy/kubernetes/kubernetes-$(OUT).yaml
	cat config/deploy/openshift/openshift.yaml config/deploy/openshift/openshift-csi.yaml > config/deploy/openshift/openshift-$(OUT).yaml

	make reset-kustomization-files

reset-kustomization-files: kustomize
	rm -f config/deploy/kubernetes/kustomization.yaml
	mkdir -p config/deploy/kubernetes
	cd config/deploy/kubernetes && $(KUSTOMIZE) create
	cd config/deploy/kubernetes && $(KUSTOMIZE) edit add base kubernetes-$(OUT).yaml

	rm -f config/deploy/openshift/kustomization.yaml
	mkdir -p config/deploy/openshift
	cd config/deploy/openshift && $(KUSTOMIZE) create
	cd config/deploy/openshift && $(KUSTOMIZE) edit add base openshift-$(OUT).yaml


# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

SERVICE_ACCOUNTS=--extra-service-accounts dynatrace-dynakube-oneagent
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-dynakube-oneagent-unprivileged
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-kubernetes-monitoring
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-routing

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: export OLM=true
bundle: export OUT=olm
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/deploy/$(PLATFORM) && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(OLM_IMAGE)
	$(KUSTOMIZE) build config/olm/$(PLATFORM) | operator-sdk generate bundle --overwrite --version $(VERSION) $(SERVICE_ACCOUNTS) $(BUNDLE_METADATA_OPTS)
	make OUT=all reset-kustomization-files
	operator-sdk bundle validate ./bundle
	rm -rf ./config/olm/$(PLATFORM)/$(VERSION)
	mkdir -p ./config/olm/$(PLATFORM)/$(VERSION)
	mv ./bundle/* ./config/olm/$(PLATFORM)/$(VERSION)
	mv ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.clusterserviceversion.yaml ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.v$(VERSION).clusterserviceversion.yaml
	mv ./bundle.Dockerfile ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	grep -v 'scorecard' ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile > ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output
	mv ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	sed 's/bundle/$(VERSION)/' ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile > ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output
	mv ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	awk '/operators.operatorframework.io.metrics.project_layout/ { print; print "  operators.operatorframework.io.bundle.channel.default.v1: alpha"; next }1' ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml >  ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml.output
	mv ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml.output ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml
	awk '/operators.operatorframework.io.$(VERSION).mediatype.v1/ { print "LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha"; print; next }1' ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile > ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output
	mv ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	echo 'LABEL com.redhat.openshift.versions="v4.7,v4.8,v4.9"' >> ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	echo 'LABEL com.redhat.delivery.operator.bundle=true' >> ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	echo 'LABEL com.redhat.delivery.backport=true' >> ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	grep -v '# Labels for testing.' ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile > ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output
	mv ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
ifeq ($(PLATFORM), openshift)
	sed 's/\bkubectl\b/oc/g' ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.v$(VERSION).clusterserviceversion.yaml > ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.v$(VERSION).clusterserviceversion.yaml.output
	mv ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.v$(VERSION).clusterserviceversion.yaml.output ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.v$(VERSION).clusterserviceversion.yaml
	echo '  com.redhat.openshift.versions: v4.7-v4.9' >> ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml
endif
	grep -v 'scorecard' ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml > ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml.output
	grep -v '  # Annotations for testing.' ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml.output > ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml
	rm ./config/olm/$(PLATFORM)/$(VERSION)/metadata/annotations.yaml.output
	mv ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.v$(VERSION).clusterserviceversion.yaml ./config/olm/$(PLATFORM)/$(VERSION)/manifests/dynatrace-operator.clusterserviceversion.yaml

.PHONY: bundle-minimal
bundle-minimal: bundle
	find ./config/olm/${PLATFORM}/${VERSION}/manifests/ ! \( -name "dynatrace-operator.v${VERSION}.clusterserviceversion.yaml" -o -name "dynatrace.com_dynakubes.yaml" \) -type f -exec rm {} +

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile -t $(BUNDLE_IMG) ./config/olm/$(PLATFORM)/

setup-pre-commit:
	$(info WARNING Make sure that golangci-lint is installed, for more info see https://golangci-lint.run/usage/install/")
	GO111MODULE=off go get github.com/daixiang0/gci
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit
