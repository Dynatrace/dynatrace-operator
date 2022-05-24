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


DYNATRACE_OPERATOR_CRD_YAML=dynatrace-operator-crd.yaml

HELM_CHART_DEFAULT_DIR=config/helm/chart/default/
HELM_GENERATED_DIR=$(HELM_CHART_DEFAULT_DIR)/generated/
HELM_TEMPLATES_DIR=$(HELM_CHART_DEFAULT_DIR)/templates/
HELM_CRD_DIR=$(HELM_TEMPLATES_DIR)/Common/crd/

MANIFESTS_DIR=config/deploy/

KUBERNETES_CORE_YAML=$(MANIFESTS_DIR)/kubernetes/kubernetes.yaml
KUBERNETES_CSIDRIVER_YAML=$(MANIFESTS_DIR)/kubernetes/kubernetes-csidriver.yaml
KUBERNETES_OLM_YAML=$(MANIFESTS_DIR)/kubernetes/kubernetes-olm.yaml
KUBERNETES_ALL_YAML=$(MANIFESTS_DIR)/kubernetes/kubernetes-all.yaml

OPENSHIFT_CORE_YAML=$(MANIFESTS_DIR)/openshift/openshift.yaml
OPENSHIFT_CSIDRIVER_YAML=$(MANIFESTS_DIR)/openshift/openshift-csidriver.yaml
OPENSHIFT_OLM_YAML=$(MANIFESTS_DIR)/openshift/openshift-olm.yaml
OPENSHIFT_ALL_YAML=$(MANIFESTS_DIR)/openshift/openshift-all.yaml

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
test: generate-crd fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

helm-test:
	./hack/helm/test.sh

helm-lint:
	./hack/helm/lint.sh

kuttl-install:
	hack/e2e/install-kuttl.sh

kuttl-all: kuttl-activegate kuttl-oneagent

kuttl-activegate: kuttl-check-mandatory-fields
	kubectl kuttl test --config kuttl/activegate/testsuite.yaml

kuttl-oneagent: kuttl-check-mandatory-fields
	kubectl kuttl test --config kuttl/oneagent/oneagent-test.yaml

kuttl-check-mandatory-fields:
	hack/do_env_variables_exist.sh "APIURL APITOKEN PAASTOKEN"

# Build manager binary
manager: generate-crd fmt vet
	go build -o bin/manager ./src/cmd/operator/

manager-amd64: export GOOS=linux
manager-amd64: export GOARCH=amd64
manager-amd64: generate-crd fmt vet
	go build -o bin/manager-amd64 ./src/cmd/operator/

# Run against the configured Kubernetes cluster in ~/.kube/config
run: export RUN_LOCAL=true
run: export POD_NAMESPACE=dynatrace
run: generate-crd fmt vet manifests
	go run ./src/cmd/operator/

install-crd: generate-crd kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall-crd: generate-crd kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests-k8s kustomize
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	cd $(MANIFESTS_DIR)/kubernetes && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build $(MANIFESTS_DIR)/kubernetes | kubectl apply -f -

# Deploy controller in the configured OpenShift cluster in ~/.kube/config
deploy-ocp: manifests-ocp kustomize
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	cd $(MANIFESTS_DIR)/openshift && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build $(MANIFESTS_DIR)/openshift | oc apply -f -

push-image:
	./hack/build/push_image.sh

push-tagged-image: export TAG=snapshot-$(shell git branch --show-current | sed "s/[^a-zA-Z0-9_-]/-/g")
push-tagged-image: push-image

# Generate manifests e.g. CRD, RBAC etc.
manifests: prepare-manifests-directory manifests-k8s manifests-ocp strip-helm-labels

strip-helm-labels:
	./hack/build/strip-helm-labels.sh \
		$(KUBERNETES_OLM_YAML) \
		$(KUBERNETES_ALL_YAML) \
		$(OPENSHIFT_OLM_YAML) \
		$(OPENSHIFT_ALL_YAML)

prepare-manifests-directory:
	find $(MANIFESTS_DIR) -type f -not -name 'kustomization.yaml' -delete

manifests-k8s: manifests-k8s-core manifests-k8s-csidriver
	cp "$(KUBERNETES_CORE_YAML)" "$(KUBERNETES_OLM_YAML)"
	cat "$(KUBERNETES_CORE_YAML)" "$(KUBERNETES_CSIDRIVER_YAML)" > "$(KUBERNETES_ALL_YAML)"

manifests-k8s-core: manifests-helm-crd kustomize
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set installCRD=true \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set operator.image="$(MASTER_IMAGE)" > "$(KUBERNETES_CORE_YAML)"

manifests-helm-crd: generate-crd
	# Build crd
	mkdir -p "$(HELM_CRD_DIR)"
	$(KUSTOMIZE) build config/crd > $(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)

	# Copy crd to CHART PATH
	mkdir -p "$(HELM_GENERATED_DIR)"
	cp "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)" "$(HELM_GENERATED_DIR)"

manifests-k8s-csidriver:
	# Generate kubernetes-csidriver.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set operator.image="$(MASTER_IMAGE)" > "$(KUBERNETES_CSIDRIVER_YAML)"

manifests-ocp: manifests-ocp-core manifests-ocp-csidriver
	cp "$(OPENSHIFT_CORE_YAML)" "$(OPENSHIFT_OLM_YAML)"
	cat "$(OPENSHIFT_CORE_YAML)" "$(OPENSHIFT_CSIDRIVER_YAML)" > "$(OPENSHIFT_ALL_YAML)"

manifests-ocp-core: manifests-helm-crd kustomize
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set installCRD=true \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > "$(OPENSHIFT_CORE_YAML)"


manifests-ocp-csidriver:
	# Generate openshift-csi.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set partial="csi" \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > "$(OPENSHIFT_CSIDRIVER_YAML)"

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

lint: fmt vet
	gci -w .
	golangci-lint run --build-tags integration,containers_image_storage_stub --timeout 300s

generate-crd: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

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
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-activegate

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: OLM=true
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q --apis-dir ./src/api/
	cd $(MANIFESTS_DIR)/$(PLATFORM) && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(OLM_IMAGE)
	$(KUSTOMIZE) build config/olm/$(PLATFORM) | operator-sdk generate bundle --overwrite --version $(VERSION) $(SERVICE_ACCOUNTS) $(BUNDLE_METADATA_OPTS)
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
	grep -v '# Labels for testing.' ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile > ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output
	mv ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile.output ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
ifeq ($(PLATFORM), openshift)
	echo 'LABEL com.redhat.openshift.versions="v4.7,v4.8,v4.9"' >> ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	echo 'LABEL com.redhat.delivery.operator.bundle=true' >> ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
	echo 'LABEL com.redhat.delivery.backport=true' >> ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile
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

test-olm:
	./hack/setup_olm_catalog.sh

setup-pre-commit:
	$(info WARNING "Make sure that golangci-lint is installed, for more info see https://golangci-lint.run/usage/install/")
	GO111MODULE=off go get github.com/daixiang0/gci
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit
