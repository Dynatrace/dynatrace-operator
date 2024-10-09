CRD_OPTIONS ?= "crd:crdVersions=v1,maxDescLen=350,ignoreUnexportedFields=true"

OLM ?= false

HELM_CHART_DEFAULT_DIR=config/helm/chart/default/
HELM_TEMPLATES_DIR=$(HELM_CHART_DEFAULT_DIR)/templates/
HELM_CRD_DIR=$(HELM_TEMPLATES_DIR)/Common/crd/

MANIFESTS_DIR=config/deploy/
RELEASE_CRD_YAML=config/deploy/dynatrace-operator-crd.yaml

KUBERNETES_CORE_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes.yaml
KUBERNETES_CSIDRIVER_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-csi.yaml
KUBERNETES_OLM_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-olm.yaml
KUBERNETES_ALL_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-all.yaml

OPENSHIFT_CORE_YAML=$(MANIFESTS_DIR)openshift/openshift.yaml
OPENSHIFT_CSIDRIVER_YAML=$(MANIFESTS_DIR)openshift/openshift-csi.yaml
OPENSHIFT_OLM_YAML=$(MANIFESTS_DIR)openshift/openshift-olm.yaml
OPENSHIFT_ALL_YAML=$(MANIFESTS_DIR)openshift/openshift-all.yaml

ifeq ($(shell echo $CHART_VERSION),)
	ifneq ($(shell git branch --show-current | grep "^release-"),)
		# if the current branch is a release branch
		ifneq ($(shell grep "^version:" $(HELM_CHART_DEFAULT_DIR)/Chart.yaml | grep snapshot),)
			CHART_VERSION=$(shell git branch --show-current | cut -d'-' -f2-).0
		else
			CHART_VERSION=
		endif
	else ifeq ($(shell git branch --show-current), main)
		# if the current branch is the main branch
		CHART_VERSION=0.0.0-snapshot
	else
		# otherwise do not change Chart.yaml
		CHART_VERSION=
	endif
endif
