IMAGE ?= quay.io/dynatrace/dynatrace-operator
DEBUG ?= false

#Needed for the e2e pipeline to work
BRANCH ?= $(shell git branch --show-current)
SNAPSHOT_SUFFIX ?= $(shell echo "${BRANCH}" | sed "s/[^a-zA-Z0-9_-]/-/g")
ifneq ($(BRANCH), main)
	TAG ?= snapshot-${SNAPSHOT_SUFFIX}
else
	TAG ?= snapshot
endif

#use the digest if digest is set
ifeq ($(DIGEST),)
	IMAGE_URI ?= "$(IMAGE):$(TAG)"
else
	IMAGE_URI ?= "$(IMAGE):$(TAG)@$(DIGEST)"
endif



ensure-tag-not-snapshot:
ifeq ($(TAG), snapshot)
	$(error "Image tag is snapshot, please set TAG to a valid tag")
endif

## Builds an Operator image with a given IMAGE and TAG
images/build: ensure-tag-not-snapshot
	./hack/build/build_image.sh "${IMAGE}" "${TAG}" "${DEBUG}"

## Build an Operator FIPS image with a give IMAGE and TAG
# because cross-compile takes ~1h, we want to build fips locally only for local architecture
# so that's why the recommended way to run it (assuming local platfrom is arm64) is `OPERATOR_DEV_BUILD_PLATFORM="linux/arm64" make images/build/fips
images/build/fips: ensure-tag-not-snapshot
	./hack/build/build_image.sh "${IMAGE}" "${TAG}-fips" "${DEBUG}" "fips.Dockerfile" "true"

## Pushes an ALREADY BUILT Operator image with a given IMAGE and TAG
images/push: ensure-tag-not-snapshot
	./hack/build/push_image.sh "${IMAGE}" "${TAG}"

## Builds an Operator image and pushes it
images/build/push: images/build images/push

## Builds and pushes the deployer image for the Google marketplace to the development environment on GCR
images/gcr/deployer:
	./hack/gcr/deployer-image.sh ":${TAG}"
