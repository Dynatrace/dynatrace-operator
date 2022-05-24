# Running 'make' runs the first rule in the makefile
# In order to print the help text in this case,
# 	'default' has been put here even before the includes
# The position of a 'default' target that prints a help screen is a
# 	makefile best-practice
## Prints help text
default: help

# Help has been copied from 'https://docs.cloudposse.com/reference/best-practices/make-best-practices/'
# What exactly it does line by line is a mystery, but the printed help text looks nice
# Basically, it takes every target, even the ones from includes, and prints their name and the comment above it which is marked by two ##
# If there is no such comment line, e.g., "## Prints a help screen", the target is not printed at all
## Prints a help screen
help:
	@printf "Available targets:\n\n"
	@awk '/^[a-zA-Z\-_0-9%:\\]+/ { \
	  helpMessage = match(lastLine, /^## (.*)/); \
	  if (helpMessage) { \
		helpCommand = $$1; \
		helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
  gsub("\\\\", "", helpCommand); \
  gsub(":+$$", "", helpCommand); \
		printf "  \x1b[32;01m%-35s\x1b[0m %s\n", helpCommand, helpMessage; \
	  } \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST) | sort -u
	@printf "\n"

SHELL ?= bash

-include hack/make/*.mk
-include hack/make/manifests/*.mk
-include hack/make/tests/*.mk
-include hack/make/deploy/*.mk

## Installs dependencies
deps: prerequisites/setup-pre-commit prerequisites/kustomize prerequisites/controller-gen

## Builds the operator image and pushes it to quay with a snapshot tag
build: images/push/tagged

## Installs (deploys) the operator on a Kubernetes cluster
install: deploy/kubernetes

## Installs dependencies, builds and pushes a tagged operator image, and deploys the operator on a cluster
all: deps build install

# Generates manifests e.g. CRD, RBAC etc, for Kubernetes and OpenShift
manifests: manifests/kubernetes manifests/openshift


