package troubleshoot

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const activeGateCheckLoggerName = "activegate"

func checkActiveGateOOMKilled(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) {
	log := baseLog.WithName(activeGateCheckLoggerName)

	logNewCheckf(log, "Checking ActiveGate pods for OOMKilled containers ...")

	podList := &corev1.PodList{}

	err := apiReader.List(ctx, podList,
		client.InNamespace(dk.Namespace),
		client.MatchingLabels{
			k8slabel.AppNameLabel:      k8slabel.ActiveGateComponentLabel,
			k8slabel.AppCreatedByLabel: dk.Name,
			k8slabel.AppManagedByLabel: version.AppName,
		},
	)
	if err != nil {
		logWarningf(log, "Failed to list ActiveGate pods: %v", err)

		return
	}

	oomKilledFound := false

	for _, pod := range podList.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if isOOMKilled(cs.LastTerminationState) {
				oomKilledFound = true

				terminated := cs.LastTerminationState.Terminated
				logWarningf(log, "pod %q: container %q was OOMKilled at %s (exit code %d)",
					pod.Name, cs.Name, terminated.FinishedAt.UTC().Format("2006-01-02 15:04:05 UTC"), terminated.ExitCode)
			}
		}
	}

	if !oomKilledFound {
		logOkf(log, "No OOMKilled ActiveGate containers found.")
	}
}

func isOOMKilled(state corev1.ContainerState) bool {
	return state.Terminated != nil && state.Terminated.Reason == "OOMKilled"
}
