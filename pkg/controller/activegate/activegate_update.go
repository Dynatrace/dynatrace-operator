package activegate

import (
	"context"
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/docker"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

func (r *ReconcileActiveGate) updatePods(pod *corev1.Pod, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (*reconcile.Result, error) {
	if !instance.Spec.DisableActivegateUpdate &&
		instance.Status.UpdatedTimestamp.Add(UpdateInterval).Before(time.Now()) {
		log.Info("checking for outdated pods")
		// Check if pods have latest activegate version
		outdatedPods, err := r.findOutdatedPods(log, instance)
		if err != nil {
			result := builder.ReconcileAfterFiveMinutes()
			// Too many requests, requeue after five minutes
			return &result, err
		}

		err = r.deletePods(log, outdatedPods)
		if err != nil {
			log.Error(err, err.Error())
			return &reconcile.Result{}, err
		}
		r.updateInstanceStatus(pod, instance, secret)
	} else if instance.Spec.DisableActivegateUpdate {
		log.Info("Skipping updating pods because of configuration", "disableActivegateUpdate", true)
	}
	return nil, nil
}

func (r *ReconcileActiveGate) findOutdatedPods(logger logr.Logger, instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error) {
	pods, err := r.findPods(instance)
	if err != nil {
		logger.Error(err, "failed to list pods")
		return nil, err
	}

	var outdatedPods []corev1.Pod
	for _, pod := range pods {
		for _, status := range pod.Status.ContainerStatuses {
			logger.Info("pods container status", "pod", pod.Name, "container", status.Name, "image id", status.ImageID)

			imagePullSecret := &corev1.Secret{}
			err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: pod.Namespace, Name: "aws-registry"}, imagePullSecret)
			if err != nil {
				logger.Error(err, err.Error())
			}

			dockerConf, err := parser.NewDockerConfig(imagePullSecret)
			if err != nil {
				logger.Info("could not parse docker config from image pull secret", "error", err.Error())
			}

			isLatest, err := isImageLatest(logger, instance, status, dockerConf)
			if err != nil {
				logger.Error(err, err.Error())
				//Error during image check, do nothing an continue with next status
				continue
			}

			logger.Info("checked image version", "latest", isLatest)

			if !isLatest {
				outdatedPods = append(outdatedPods, pod)
				// Pod is outdated, break loop
				break
			}

			// If digests are the same, everything is fine
		}
	}

	return outdatedPods, nil
}

func isImageLatest(logger logr.Logger, instance *dynatracev1alpha1.ActiveGate, status corev1.ContainerStatus, dockerConfig *parser.DockerConfig) (bool, error) {
	if dockerConfig == nil {
		return false, fmt.Errorf("docker config must not be nil")
	}

	//image := instance.Spec.Image
	//if strings.TrimSpace(image) == "" {
	//	image = Image
	//}

	registry := docker.RegistryFromImage(status.Image)

	digest := strings.Split(status.ImageID, "@")[1]
	authServerName := fmt.Sprintf("https://%s", registry.Server)
	authServer, hasAuthServer := dockerConfig.Auths[authServerName]

	if !hasAuthServer {
		log.Info("could not find credentials for auth server in docker config")
		authServer = struct {
			Username string
			Password string
		}{Username: "", Password: ""}
	}

	registry.Username = authServer.Username
	registry.Password = authServer.Password

	latestManifest, err := registry.GetLatestManifest()
	if err != nil {
		logger.Info("could not fetch latest image manifest", "error", err)
		return false, err
	}

	currentManifest, err := registry.GetManifest(digest)
	if err != nil {
		logger.Info("could not fetch image manifest for digest", "digest", digest, "error", err)
		return false, err
	}

	logger.Info("Retrieved digests", "latest", latestManifest.Config.Digest, digest, currentManifest.Config.Digest)
	return latestManifest.Config.Digest == currentManifest.Config.Digest, nil
}

func (r *ReconcileActiveGate) findPods(instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.Labels),
	}
	err := r.client.List(context.TODO(), podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

const (
	Image = "activegate"
)
