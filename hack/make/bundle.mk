# Current Operator version
VERSION ?= 0.0.1
# Default platform for bundles
PLATFORM ?= openshift
# Needed variable for manifest generation
OLM ?= false
# Default bundle image with tag
BUNDLE_IMG ?= $(REGISTRY)/dynatrace/dynatrace-operator-bundle:$(VERSION)

# Options for 'bundle'
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
	@git restore config/

.PHONY: bundle/build
## Build the bundle image
bundle/build:
	cd config/olm/$(PLATFORM)/$(VERSION) && docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle/push
## Push the bundle image
bundle/push:
	docker push $(BUNDLE_IMG)

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
