package oneagent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileOneAgent) reconcileVersion(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, webhookInjection bool, dtc dtclient.Client) (bool, error) {
	if instance.Status.OneAgent.UseImmutableImage {
		return r.reconcileVersionImmutableImage(ctx, instance, fs, dtc)
	} else {
		return r.reconcileVersionInstaller(ctx, logger, instance, fs, dtc)
	}
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

	var waitSecs uint16 = 300
	if fs.WaitReadySeconds != nil {
		waitSecs = *fs.WaitReadySeconds
	}

	if len(podsToDelete) > 0 {
		if instance.Status.SetPhase(dynatracev1alpha1.Deploying) {
			err := r.updateCR(ctx, instance)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to set phase to %s", dynatracev1alpha1.Deploying))
			}
		}
	}

	// restart daemonset
	err = r.deletePods(logger, podsToDelete, buildLabels(instance.GetName()), waitSecs)
	if err != nil {
		logger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) reconcileVersionImmutableImage(ctx context.Context, instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, dtc dtclient.Client) (bool, error) {
	updateCR := false
	var waitSecs uint16 = 300
	if fs.WaitReadySeconds != nil {
		waitSecs = *fs.WaitReadySeconds
	}

	if instance.Spec.OneAgent.AutoUpdate == nil || *instance.Spec.OneAgent.AutoUpdate == true {
		r.logger.Info("checking for outdated pods")

		// Check if pods have latest agent version
		outdatedPods, err := r.findOutdatedPodsImmutableImage(ctx, r.logger, instance, isLatest())
		if err != nil {
			return updateCR, err
		}
		if len(outdatedPods) > 0 {
			updateCR = true
			err = r.deletePods(r.logger, outdatedPods, buildLabels(instance.GetName()), waitSecs)
			if err != nil {
				r.logger.Error(err, err.Error())
				return updateCR, err
			}
			instance.Status.UpdatedTimestamp = metav1.Now()

			err = r.setVersionByIP(ctx, instance, dtc)
			if err != nil {
				r.logger.Error(err, err.Error())
				return updateCR, err
			}
		}
	} else {
		r.logger.Info("Skipping updating pods because of configuration", "disableOneAgentUpdate", true)
	}
	return updateCR, nil
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

func (r *ReconcileOneAgent) findOutdatedPodsImmutableImage(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.DynaKube, isLatestFn func(logger logr.Logger, image string, imageID string, imagePullSecret *corev1.Secret) (bool, error)) ([]corev1.Pod, error) {
	pods, err := r.findPods(ctx, instance)
	if err != nil {
		logger.Error(err, "failed to list pods")
		return nil, err
	}

	var outdatedPods []corev1.Pod
	for _, pod := range pods {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Image == "" || status.ImageID == "" {
				// If image is not yet pulled skip check
				continue
			}
			logger.Info("pods container status", "pod", pod.Name, "container", status.Name, "imageID", status.ImageID)

			imagePullSecret := &corev1.Secret{}
			pullSecretName := instance.GetName() + "-pull-secret"
			if instance.Spec.CustomPullSecret != "" {
				pullSecretName = instance.Spec.CustomPullSecret
			}

			err := r.client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pullSecretName}, imagePullSecret)
			if err != nil {
				return nil, err
			}

			isLatest, err := isLatestFn(logger, status.Image, status.ImageID, imagePullSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to verify if Pod is outdated: %w", err)
			}

			if !isLatest {
				logger.Info("pod is outdated", "name", pod.Name)
				outdatedPods = append(outdatedPods, pod)
				// Pod is outdated, break loop
				break
			}
		}
	}

	return outdatedPods, err
}

func isLatest() func(logr.Logger, string, string, *corev1.Secret) (bool, error) {
	failed := false
	cache := map[string]bool{}

	return func(logger logr.Logger, image string, imageID string, imagePullSecret *corev1.Secret) (bool, error) {
		if failed {
			return true, nil
		}

		if latest, ok := cache[imageID]; ok {
			return latest, nil
		}

		logger.Info("Fetching image hash")

		dockerConfig, err := utils.NewDockerConfig(imagePullSecret)
		if err != nil {
			failed = true
			logger.Info(err.Error())
			return true, nil
		}

		dockerVersionChecker := utils.NewDockerVersionChecker(image, imageID, dockerConfig)
		latest, err := dockerVersionChecker.IsLatest()
		if err != nil {
			failed = true
			logger.Info(err.Error())
			return true, nil
		}

		cache[imageID] = latest
		return latest, nil
	}
}

func (r *ReconcileOneAgent) findPods(ctx context.Context, instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(buildLabels(instance.GetName())),
	}
	err := r.client.List(ctx, podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (r *ReconcileOneAgent) setVersionByIP(ctx context.Context, instance *dynatracev1alpha1.DynaKube, dtc dtclient.Client) error {
	pods, err := r.findPods(ctx, instance)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			return err
		}
		instance.Status.OneAgent.Version = ver
	}
	return nil
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
