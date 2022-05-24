CRD_OPTIONS ?= "crd:preserveUnknownFields=false, crdVersions=v1"
OLM ?= false

# "OTHERS" = ClusterRole, ClusterRoleBinding, Deployment, MutatingWebhookConfiguration, Role, RoleBinding, Service, ServiceAccount, ValidatingWebhookConfiguration
KUBERNETES_OTHERS_YAML=config/deploy/kubernetes/kubernetes-others.yaml
KUBERNETES_CRD_AND_OTHERS_YAML=config/deploy/kubernetes/kubernetes.yaml
KUBERNETES_CSIDRIVER_YAML=config/deploy/kubernetes/kubernetes-csidriver.yaml
KUBERNETES_OLM_YAML=config/deploy/kubernetes/kubernetes-olm.yaml
KUBERNETES_ALL_YAML=config/deploy/kubernetes/kubernetes-all.yaml

OPENSHIFT_OTHERS_YAML=config/deploy/openshift/openshift-others.yaml
OPENSHIFT_CRD_AND_OTHERS_YAML=config/deploy/openshift/openshift.yaml
OPENSHIFT_CSIDRIVER_YAML=config/deploy/openshift/openshift-csidriver.yaml
OPENSHIFT_OLM_YAML=config/deploy/openshift/openshift-olm.yaml
OPENSHIFT_ALL_YAML=config/deploy/openshift/openshift-all.yaml
