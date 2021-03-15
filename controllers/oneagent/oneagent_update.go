package oneagent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileOneAgent) reconcileVersion(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, dtc dtclient.Client) (bool, error) {
	// Immutable images are updated by attaching the OneAgent version to the the Pod Spec as an annotation.
	if !instance.Status.OneAgent.UseImmutableImage {
		return r.reconcileVersionInstaller(ctx, logger, instance, fs, dtc)
	}
	return false, nil
}

func (r *ReconcileOneAgent) reconcileVersionInstaller(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, dtc dtclient.Client) (bool, error) {
	updateCR := false

	desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return false, fmt.Errorf("failed to get desired version: %w", err)
	} else if desired != "" && desired != instance.Status.OneAgent.Version {
		instance.Status.OneAgent.Version = desired
		updateCR = true
		if isDesiredNewer(instance.Status.OneAgent.Version, desired, logger) {
			logger.Info("new version available", "actual", instance.Status.OneAgent.Version, "desired", desired)
		}
	}

	podList, err := r.findPods(ctx, instance)
	if err != nil {
		logger.Error(err, "failed to list pods", "podList", podList)
		return updateCR, err
	}

	podsToDelete, err := findOutdatedPodsInstaller(podList, dtc, instance, logger)
	if err != nil {
		return updateCR, err
	}

	waitSecs := getWaitReadySeconds(fs)

	// restart daemonset
	err = r.deletePods(logger, podsToDelete, buildLabels(instance.GetName(), r.feature), waitSecs)
	if err != nil {
		logger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func getWaitReadySeconds(fs *dynatracev1alpha1.FullStackSpec) uint16 {
	if fs.WaitReadySeconds != nil {
		return *fs.WaitReadySeconds
	}
	return 300
}

// findOutdatedPodsInstaller determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func findOutdatedPodsInstaller(pods []corev1.Pod, dtc dtclient.Client, instance *dynatracev1alpha1.DynaKube, logger logr.Logger) ([]corev1.Pod, error) {
	var doomedPods []corev1.Pod

	for _, pod := range pods {
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			err = handleAgentVersionForIPError(err, instance, pod, nil)
			if err != nil {
				return doomedPods, err
			}
		} else {
			if isDesiredNewer(ver, instance.Status.OneAgent.Version, logger) {
				doomedPods = append(doomedPods, pod)
			}
		}
	}

	return doomedPods, nil
}

func (r *ReconcileOneAgent) findPods(ctx context.Context, instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(buildLabels(instance.GetName(), r.feature)),
	}
	err := r.client.List(ctx, podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func isDesiredNewer(actual string, desired string, logger logr.Logger) bool {
	aa := strings.Split(actual, ".")
	da := strings.Split(desired, ".")

	for i := 0; i < len(aa); i++ {
		if i == len(aa)-1 {
			if aa[i] < da[i] {
				return true
			} else if aa[i] > da[i] {
				var err = errors.New("downgrade error")
				logger.Error(err, "downgrade detected! downgrades are not supported")
				return false
			} else {
				return false
			}
		}

		av, err := strconv.Atoi(aa[i])
		if err != nil {
			logger.Error(err, "failed to parse actual version number", "actual", actual)
			return false
		}

		dv, err := strconv.Atoi(da[i])
		if err != nil {
			logger.Error(err, "failed to parse desired version number", "desired", desired)
			return false
		}

		if av < dv {
			return true
		}
		if av > dv {
			var err = errors.New("downgrade error")
			logger.Error(err, "downgrade detected! downgrades are not supported")
			return false
		}
	}

	return false
}
