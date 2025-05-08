package oneagent

import (
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	corev1 "k8s.io/api/core/v1"
)

type installerInfo struct {
	flavor       string
	technologies string
	installPath  string
	installerURL string
	version      string
}

func getInstallerInfo(pod *corev1.Pod, dk dynakube.DynaKube) installerInfo {
	return installerInfo{
		flavor:       maputils.GetField(pod.Annotations, AnnotationFlavor, ""),
		technologies: url.QueryEscape(maputils.GetField(pod.Annotations, oacommon.AnnotationTechnologies, "all")),
		installPath:  maputils.GetField(pod.Annotations, oacommon.AnnotationInstallPath, oacommon.DefaultInstallPath),
		installerURL: maputils.GetField(pod.Annotations, AnnotationInstallerUrl, ""),
		version:      dk.OneAgent().GetCodeModulesVersion(),
	}
}
