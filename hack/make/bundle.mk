# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif

# Default platform for bundles
PLATFORM ?= openshift
# Needed variable for manifest generation
OLM ?= false
# Default bundle image with tag
BUNDLE_IMG ?= $(REGISTRY)/$(REPOSITORY)/dynatrace-operator-bundle:$(VERSION)

# CONTAINER_TOOL defines the container tool to be used for building images.
ifneq ($(shell command -v podman),)
	CONTAINER_TOOL ?= podman
else
	CONTAINER_TOOL ?= docker
endif

.PHONY: bundle
## Generates bundle manifests and metadata, then validates generated files
bundle: manifests/$(PLATFORM)/core
	./hack/build/bundle.sh "$(PLATFORM)" "$(VERSION)" "$(BUNDLE_CHANNELS)" "$(BUNDLE_DEFAULT_CHANNEL)"
	@git restore config/

.PHONY: bundle/build
## Build the bundle image
bundle/build:
	cd config/olm/$(PLATFORM)/$(VERSION) && $(CONTAINER_TOOL) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle/push
## Push the bundle image
bundle/push:
	$(CONTAINER_TOOL) push $(BUNDLE_IMG)

.PHONY: bundle/install
## Deploy the bundle
bundle/install: bundle/build bundle/push
	operator-sdk run bundle $(BUNDLE_IMG) --namespace dynatrace --timeout 5m

.PHONY: bundle/upgrade
## Upgrade previously installed bundle
bundle/upgrade: bundle/build bundle/push
	operator-sdk run bundle-upgrade $(BUNDLE_IMG) --namespace dynatrace --timeout 5m

.PHONY: bundle/cleanup
## Clean up bundle
bundle/cleanup:
	operator-sdk cleanup dynatrace-operator --delete-all --delete-crds --delete-operator-groups --namespace dynatrace --timeout 5m
