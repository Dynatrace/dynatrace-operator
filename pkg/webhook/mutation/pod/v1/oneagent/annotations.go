package oneagent

import (
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
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

func getInstallerInfo(pod *corev1.Pod, dk dynakube.DynaKube) installerInfo {
	return installerInfo{
		flavor:       maputils.GetField(pod.Annotations, dtwebhook.AnnotationFlavor, ""),
		technologies: url.QueryEscape(maputils.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all")),
		installPath:  maputils.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath),
		installerURL: maputils.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, ""),
		version:      dk.OneAgent().GetCodeModulesVersion(),
	}
}
