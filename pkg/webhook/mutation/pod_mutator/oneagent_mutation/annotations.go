package oneagent_mutation

import (
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	_map "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
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

func setNotInjectedAnnotations(pod *corev1.Pod, reason string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[dtwebhook.AnnotationOneAgentInjected] = "false"
	pod.Annotations[dtwebhook.AnnotationOneAgentReason] = reason
}

func getInstallerInfo(pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) installerInfo {
	return installerInfo{
		flavor:       _map.GetField(pod.Annotations, dtwebhook.AnnotationFlavor, ""),
		technologies: url.QueryEscape(_map.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all")),
		installPath:  _map.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath),
		installerURL: _map.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, ""),
		version:      dynakube.CodeModulesVersion(),
	}
}
