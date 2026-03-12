REGISTRY ?= ghcr.io
REPOSITORY ?= dynatrace
IMAGE ?= "$(REGISTRY)/$(REPOSITORY)/dynatrace-operator"
DEBUG ?= false
OPERATOR_BUILD_PLATFORM ?= linux/amd64
OPERATOR_BUILD_ARCH ?= $(shell echo "${OPERATOR_BUILD_PLATFORM}" | sed "s/.*\///")

#Needed for the e2e pipeline to work
BRANCH ?= $(shell git branch --show-current)
TAG_BRANCH_SUFFIX ?= $(shell hack/build/ci/sanitize-branch-name.sh "${BRANCH}")
ifneq ($(BRANCH), main)
	TAG ?= snapshot-${TAG_BRANCH_SUFFIX}
else
	TAG ?= snapshot
endif

FIPS_TAG ?= ${TAG}-fips

#use the digest if digest is set
ifeq ($(DIGEST),)
	IMAGE_URI ?= "$(IMAGE):$(TAG)"
else
	IMAGE_URI ?= "$(IMAGE):$(TAG)@$(DIGEST)"
endif

BUILD_IMAGE_SH := ./hack/build/build_image.sh
PUSH_IMAGE_SH := ./hack/build/push_image.sh
CREATE_IMAGE_INDEX_SH := ./hack/build/ci/create-image-index.sh

ensure-tag-not-snapshot:
ifeq ($(TAG), snapshot)
	$(error "Image tag is snapshot, please set TAG to a valid tag")
endif

## Builds an Operator image with a given IMAGE and TAG-OPERATOR_BUILD_ARCH
images/build: ensure-tag-not-snapshot
	$(BUILD_IMAGE_SH) "${IMAGE}" "${TAG}-${OPERATOR_BUILD_ARCH}" "${DEBUG}" "Dockerfile" "${OPERATOR_BUILD_PLATFORM}"

## Pushes an ALREADY BUILT Operator image as an image-index with a given IMAGE and TAG, containing the image for the OPERATOR_BUILD_ARCH
images/push: ensure-tag-not-snapshot
	$(PUSH_IMAGE_SH) "${IMAGE}" "${TAG}-${OPERATOR_BUILD_ARCH}"
	$(CREATE_IMAGE_INDEX_SH) "${IMAGE}:${TAG}" "${OPERATOR_BUILD_ARCH}"

## Builds an Operator image and pushes it
images/build/push: images/build images/push

## Build an Operator FIPS image with a give IMAGE and TAG
# because cross-compile takes ~1h, we want to build fips locally only for local architecture
# so that's why the recommended way to run it (assuming local platfrom is arm64) is `OPERATOR_DEV_BUILD_PLATFORM="linux/arm64" make images/build/fips
images/build/fips: ensure-tag-not-snapshot
	$(BUILD_IMAGE_SH) "${IMAGE}" "${FIPS_TAG}" "${DEBUG}" "fips.Dockerfile" "${OPERATOR_BUILD_PLATFORM}"

images/push/fips: ensure-tag-not-snapshot
	$(PUSH_IMAGE_SH) "${IMAGE}" "${FIPS_TAG}-${OPERATOR_BUILD_ARCH}"
	$(CREATE_IMAGE_INDEX_SH) "${IMAGE}:${FIPS_TAG}" "${OPERATOR_BUILD_ARCH}"

images/build/push/fips: images/build/fips images/push/fips

images/build/multi: ensure-tag-not-snapshot
	$(MAKE) OPERATOR_BUILD_PLATFORM="linux/amd64" images/build
	$(MAKE) OPERATOR_BUILD_PLATFORM="linux/arm64" images/build

images/push/multi: ensure-tag-not-snapshot
	$(PUSH_IMAGE_SH) "${IMAGE}" "${TAG}-arm64"
	$(PUSH_IMAGE_SH) "${IMAGE}" "${TAG}-amd64"
	$(CREATE_IMAGE_INDEX_SH) "${IMAGE}:${TAG}" "arm64,amd64"

images/build/push/multi: images/build/multi images/push/multi

## Builds and pushes the deployer image for the Google marketplace to the development environment on GCR
images/gcr/deployer:
	./hack/gcr/deployer-image.sh ":${TAG}"
