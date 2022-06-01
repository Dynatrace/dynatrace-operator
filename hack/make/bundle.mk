-include images.mk
-include prerequisites.mk
-include manifests/*.mk

# Current Operator version
VERSION ?= 0.0.1
# Default platform for bundles
PLATFORM ?= openshift
# Needed variable for manifest generation
OLM ?= false

SERVICE_ACCOUNTS=--extra-service-accounts dynatrace-dynakube-oneagent
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-dynakube-oneagent-unprivileged
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-kubernetes-monitoring
SERVICE_ACCOUNTS+=--extra-service-accounts dynatrace-activegate

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)


.PHONY: bundle
## Generates bundle manifests and metadata, then validates generated files
bundle: OLM=true
bundle: manifests/kubernetes manifests/openshift prerequisites/kustomize
	operator-sdk generate kustomize manifests -q --apis-dir ./src/api/
	cd config/deploy/$(PLATFORM) && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(OLM_IMAGE)
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

.PHONY: bundle/minimal
## Generates bundle manifests and metadata, validates generated files and removes everything but the CSV file
bundle/minimal: bundle
	find ./config/olm/${PLATFORM}/${VERSION}/manifests/ ! \( -name "dynatrace-operator.v${VERSION}.clusterserviceversion.yaml" -o -name "dynatrace.com_dynakubes.yaml" \) -type f -exec rm {} +

.PHONY: bundle/build
## Builds the docker image used for OLM deployment
bundle/build:
	docker build -f ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile -t $(BUNDLE_IMG) ./config/olm/$(PLATFORM)/
