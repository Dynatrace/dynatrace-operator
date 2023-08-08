CRD_OPTIONS ?= "crd:crdVersions=v1"
OLM ?= false

HELM_CHART_DEFAULT_DIR=config/helm/chart/default/
HELM_TEMPLATES_DIR=$(HELM_CHART_DEFAULT_DIR)/templates/
HELM_CRD_DIR=$(HELM_TEMPLATES_DIR)/Common/crd/

MANIFESTS_DIR=config/deploy/
RELEASE_CRD_YAML=config/deploy/dynatrace-operator-crd.yaml

KUBERNETES_CORE_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes.yaml
KUBERNETES_AUTOPILOT_YAML=$(MANIFESTS_DIR)kubernetes/gke-autopilot.yaml
KUBERNETES_CSIDRIVER_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-csi.yaml
KUBERNETES_OLM_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-olm.yaml
KUBERNETES_ALL_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-all.yaml

OPENSHIFT_CORE_YAML=$(MANIFESTS_DIR)openshift/openshift.yaml
OPENSHIFT_CSIDRIVER_YAML=$(MANIFESTS_DIR)openshift/openshift-csi.yaml
OPENSHIFT_OLM_YAML=$(MANIFESTS_DIR)openshift/openshift-olm.yaml
OPENSHIFT_ALL_YAML=$(MANIFESTS_DIR)openshift/openshift-all.yaml

ifneq ($(shell echo "$(CHART_VERSION_VAR)") | grep "v",)
	# if the current branch is a release branch
	ifneq ($(shell grep "^version:" $(HELM_CHART_DEFAULT_DIR)/Chart.yaml) | grep "snapshot",)
		CHART_VERSION=$(CHART_VERSION_VAR)
	else
		CHART_VERSION=
	endif
else ifneq ($(shell echo "$(CHART_VERSION_VAR)" | grep "main"),)
	# if the current branch is the main branch
	CHART_VERSION=0.0.0-snapshot
else
	# otherwise do not change Chart.yaml
	CHART_VERSION=
endif
