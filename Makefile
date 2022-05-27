# Running 'make' runs the first rule in the makefile
# In order to print the help text in this case,
# 	'default' has been put here even before the includes
# The position of a 'default' target that prints a help screen is a
# 	makefile best-practice
## Prints help text
default: help

# Help has been copied from 'https://docs.cloudposse.com/reference/best-practices/make-best-practices/'
# Basically, it takes every target, even the ones from includes, and prints their name and the comment above it which is marked by two ##
# If there is no such comment line, e.g., "## Prints a help screen", the target is not printed at all
#
# Code breakdown:
# '/^[a-zA-Z\-_0-9%:\\]+/ Match every line that is neither a comment nor a command, i.e. only targets
# helpMessage = match(lastLine, /^## (.*)/); If `lastLine` starts with `## ` it is assumed to be a help message
# if (helpMessage) { If it is indeed a help message
# 	helpCommand = $$1; Then the command it describes is the next line
#   helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \  Remove the `## ` from the help comment
#   gsub("\\\\", "", helpCommand); \ Remove `\\` from the command string
#   gsub(":+$$", "", helpCommand); \ Remove the colon and everything past it from the command string
#   printf
#  		"  \x1b[32;01m Escape code to set the output color to green
#		%-35s Print the first argument and truncate to 35 characters
#  		\x1b[0m Reset the output color
#		%s\n", helpCommand, helpMessage; Print the second argument, supply command and message as arguments
# { lastLine = $$0 }' Iterates through every line and assigns it to `lastLine`
# $(MAKEFILE_LIST) Holds the filenames to every Makefile so `awk` can iterate through it
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


