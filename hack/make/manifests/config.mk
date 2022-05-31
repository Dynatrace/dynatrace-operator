CRD_OPTIONS ?= "crd:crdVersions=v1"
OLM ?= false

DYNATRACE_OPERATOR_CRD_YAML=dynatrace-operator-crd.yaml

HELM_CHART_DEFAULT_DIR=config/helm/chart/default/
HELM_GENERATED_DIR=$(HELM_CHART_DEFAULT_DIR)/generated/
HELM_TEMPLATES_DIR=$(HELM_CHART_DEFAULT_DIR)/templates/
HELM_CRD_DIR=$(HELM_TEMPLATES_DIR)/Common/crd/

MANIFESTS_DIR=config/deploy/

KUBERNETES_CORE_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes.yaml
KUBERNETES_CSIDRIVER_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-csidriver.yaml
KUBERNETES_OLM_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-olm.yaml
KUBERNETES_ALL_YAML=$(MANIFESTS_DIR)kubernetes/kubernetes-all.yaml

OPENSHIFT_CORE_YAML=$(MANIFESTS_DIR)openshift/openshift.yaml
OPENSHIFT_CSIDRIVER_YAML=$(MANIFESTS_DIR)openshift/openshift-csidriver.yaml
OPENSHIFT_OLM_YAML=$(MANIFESTS_DIR)openshift/openshift-olm.yaml
OPENSHIFT_ALL_YAML=$(MANIFESTS_DIR)openshift/openshift-all.yaml