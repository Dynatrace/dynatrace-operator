package troubleshoot

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const activeGateCheckLoggerName = "activegate"

func checkActiveGates(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, dk *dynakube.DynaKube) error {
	log := baseLog.WithName(activeGateCheckLoggerName)

	err := checkActiveGateOOM(ctx, log, apiReader, dk)
	if err != nil {
		logErrorf(log, "Failed to check ActiveGate pods: %v", err)

		return err
	}

	return nil
}

func checkActiveGateOOM(ctx context.Context, log logr.Logger, apiReader client.Reader, dk *dynakube.DynaKube) error {
	logNewCheckf(log, "Checking ActiveGate pods")

	labels := map[string]string{
		k8slabel.AppNameLabel:      k8slabel.ActiveGateComponentLabel,
		k8slabel.AppCreatedByLabel: dk.Name,
		k8slabel.AppManagedByLabel: version.AppName,
	}

	return checkOOMKilled(ctx, log, apiReader, dk.Namespace, labels)
}

func checkOOMKilled(ctx context.Context, log logr.Logger, apiReader client.Reader, namespace string, labels map[string]string) error {
	podList := &corev1.PodList{}

	err := apiReader.List(ctx, podList,
		client.InNamespace(namespace),
		client.MatchingLabels(labels),
	)
	if err != nil {
		logWarningf(log, "Failed to list pods: %v", err)

		return err
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
		logOkf(log, "No OOMKilled containers found.")
	}

	return nil
}

func isOOMKilled(state corev1.ContainerState) bool {
	return state.Terminated != nil && state.Terminated.Reason == "OOMKilled"
}
