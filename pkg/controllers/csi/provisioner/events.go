package csiprovisioner

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

type updaterEventRecorder struct {
	dk       *dynakube.DynaKube
	recorder record.EventRecorder
}

func (event *updaterEventRecorder) sendFailedInstallAgentVersionEvent(version, tenantUUID string) {
	event.recorder.Eventf(event.dk,
		corev1.EventTypeWarning,
		failedInstallAgentVersionEvent,
		"Failed to install agent version: %s to tenant: %s", version, tenantUUID)
}

func (event *updaterEventRecorder) sendInstalledAgentVersionEvent(version, tenantUUID string) {
	event.recorder.Eventf(event.dk,
		corev1.EventTypeNormal,
		installAgentVersionEvent,
		"Installed agent version: %s to tenant: %s", version, tenantUUID)
}
