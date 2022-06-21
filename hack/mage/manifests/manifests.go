package manifests

import (
	"github.com/Dynatrace/dynatrace-operator/hack/mage/config"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/crd"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/prerequisites"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// KubernetesCsi generates a manifest for Kubernetes solely for a CSI driver deployment
func KubernetesCsi() error {
	// Generate kubernetes-csidriver.yaml
	olm := "false"
	if config.OLM {
		olm = "true"
	}
	return sh.Run("/bin/bash", "-c", "helm template dynatrace-operator config/helm/chart/default --namespace dynatrace --set partial=\"csi\" --set platform=\"kubernetes\" --set manifests=true --set olm=\""+olm+"\" --set image=\""+config.MASTER_IMAGE+"\" > \""+config.KUBERNETES_CSIDRIVER_YAML+"\"")
}

// KubernetesCore generates an Kubernetes manifest with a CRD
func KubernetesCore() error {
	mg.SerialDeps(crd.CrdHelm, prerequisites.Kustomize)
	olm := "false"
	if config.OLM {
		olm = "true"
	}
	return sh.Run("/bin/bash", "-c", "helm template dynatrace-operator config/helm/chart/default --namespace dynatrace --set installCRD=true --set platform=\"kubernetes\" --set manifests=true --set olm=\""+olm+"\" --set image=\""+config.MASTER_IMAGE+"\" > \""+config.KUBERNETES_CORE_YAML+"\"")
}

// Kubernetes generates a manifest for Kubernetes including a CRD, a CSI driver deployment and a OLM version
func Kubernetes() error {
	mg.SerialDeps(config.Init, KubernetesCore, KubernetesCsi)

	err := sh.Run("cp", config.KUBERNETES_CORE_YAML, config.KUBERNETES_OLM_YAML)
	if err != nil {
		return nil
	}
	return sh.Run("/bin/bash", "-c", "cat \""+config.KUBERNETES_CORE_YAML+"\" \""+config.KUBERNETES_CSIDRIVER_YAML+"\" > \""+config.KUBERNETES_ALL_YAML+"\"")
}

// OpenshiftCsi generates a manifest for Openshift solely for a CSI driver deployment
func OpenshiftCsi() error {
	// Generate openshift-csi.yaml
	olm := "false"
	if config.OLM {
		olm = "true"
	}
	return sh.Run("/bin/bash", "-c", "helm template dynatrace-operator config/helm/chart/default --namespace dynatrace --set partial=\"csi\" --set platform=\"openshift\" --set manifests=true --set olm=\""+olm+"\" --set createSecurityContextConstraints=\"true\" --set image=\""+config.MASTER_IMAGE+"\" > \""+config.OPENSHIFT_CSIDRIVER_YAML+"\"")
}

// OpenshiftCore generates an OpenShift manifest with a CRD
func OpenshiftCore() error {
	mg.SerialDeps(crd.CrdHelm, prerequisites.Kustomize)
	olm := "false"
	if config.OLM {
		olm = "true"
	}
	return sh.Run("/bin/bash", "-c", "helm template dynatrace-operator config/helm/chart/default --namespace dynatrace --set installCRD=true --set platform=\"openshift\" --set manifests=true --set olm=\""+olm+"\" --set createSecurityContextConstraints=\"true\" --set image=\""+config.MASTER_IMAGE+"\" > \""+config.OPENSHIFT_CORE_YAML+"\"")
}

// Openshift generates a manifest for OpenShift including a CRD and a CSI driver deployment
func Openshift() error {
	mg.SerialDeps(config.Init, OpenshiftCore, OpenshiftCsi)

	err := sh.Run("cp", config.OPENSHIFT_CORE_YAML, config.OPENSHIFT_OLM_YAML)
	if err != nil {
		return nil
	}
	return sh.Run("/bin/bash", "-c", "cat \""+config.OPENSHIFT_CORE_YAML+"\" \""+config.OPENSHIFT_CSIDRIVER_YAML+"\" > \""+config.OPENSHIFT_ALL_YAML+"\"")
}
