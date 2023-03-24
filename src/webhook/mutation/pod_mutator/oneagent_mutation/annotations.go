package oneagent_mutation

import (
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

type installerInfo struct {
	flavor       string
	technologies string
	installPath  string
	installerURL string
	version      string
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[dtwebhook.AnnotationOneAgentInjected] = "true"
}

func getInstallerInfo(pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) installerInfo {
	return installerInfo{
		flavor:       kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFlavor, ""),
		technologies: url.QueryEscape(kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all")),
		installPath:  kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath),
		installerURL: kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, ""),
		version:      dynakube.CodeModulesVersion(),
	}
}
