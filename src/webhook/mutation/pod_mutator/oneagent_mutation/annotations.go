package oneagent_mutation

import (
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

type installerInfo struct {
	flavor        string
	technologies  string
	installPath   string
	installerURL  string
	failurePolicy string
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[dtwebhook.AnnotationOneAgentInjected] = "true"
}

func getInstallerInfo(pod *corev1.Pod) installerInfo {
	return installerInfo{
		flavor:        kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFlavor, arch.FlavorMultidistro),
		technologies:  url.QueryEscape(kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all")),
		installPath:   kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath),
		installerURL:  kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, ""),
		failurePolicy: kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent"),
	}
}
