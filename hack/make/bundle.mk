# Current Operator version
VERSION ?= 0.0.1
# Default platform for bundles
PLATFORM ?= openshift
# Needed variable for manifest generation
OLM ?= false
# Default bundle image with tag
BUNDLE_IMG ?= quay.io/dynatrace/olm_catalog_tests:$(VERSION)

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

bundle/build:
	podman build -f ./bundle.Dockerfile -t $(BUNDLE_IMG)
	podman push $(BUNDLE_IMG)

bundle/run: bundle bundle/build
	operator-sdk run bundle $(BUNDLE_IMG)

bundle/cleanup:
	operator-sdk cleanup dynatrace-operator
