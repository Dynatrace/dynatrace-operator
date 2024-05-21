package csiprovisioner

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

type updaterEventRecorder struct {
	dynakube *dynatracev1beta2.DynaKube
	recorder record.EventRecorder
}

func (event *updaterEventRecorder) sendFailedInstallAgentVersionEvent(version, tenantUUID string) {
	event.recorder.Eventf(event.dynakube,
		corev1.EventTypeWarning,
		failedInstallAgentVersionEvent,
		"Failed to install agent version: %s to tenant: %s", version, tenantUUID)
}

func (event *updaterEventRecorder) sendInstalledAgentVersionEvent(version, tenantUUID string) {
	event.recorder.Eventf(event.dynakube,
		corev1.EventTypeNormal,
		installAgentVersionEvent,
		"Installed agent version: %s to tenant: %s", version, tenantUUID)
}
