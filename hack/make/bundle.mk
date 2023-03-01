# Current Operator version
VERSION ?= 0.0.1
# Default platform for bundles
PLATFORM ?= openshift
# Needed variable for manifest generation
OLM ?= false
# Default bundle image with tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif

.PHONY: bundle
## Generates bundle manifests and metadata, then validates generated files
bundle: manifests/$(PLATFORM)
	./hack/build/bundle.sh "$(PLATFORM)" "$(VERSION)" "$(BUNDLE_CHANNELS)" "$(BUNDLE_DEFAULT_CHANNEL)"

.PHONY: bundle/minimal
## Generates bundle manifests and metadata, validates generated files and removes everything but the CSV file
bundle/minimal: bundle
	find ./config/olm/${PLATFORM}/${VERSION}/manifests/ ! \( -name "dynatrace-operator.v${VERSION}.clusterserviceversion.yaml" -o -name "dynatrace.com_dynakubes.yaml" \) -type f -exec rm {} +

.PHONY: bundle/build
## Builds the docker image used for OLM deployment
bundle/build:
	docker build -f ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile -t $(BUNDLE_IMG) ./config/olm/$(PLATFORM)/
