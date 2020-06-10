package activegate

import (
	"context"
	"encoding/json"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
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
	}
	return nil, nil
}

func (r *ReconcileActiveGate) findOutdatedPods(logger logr.Logger, instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error) {
	//secret, err := r.getTokenSecret(instance)
	//if err != nil {
	//	logger.Error(err, "failed to retrieve token secret")
	//	return nil, err
	//}

	//dtClient, err := builder.BuildDynatraceClient(r.client, instance, secret)
	//if err != nil {
	//	logger.Error(err, err.Error())
	//	return nil, err
	//}

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

			config, hasConfig := imagePullSecret.Data[".dockerconfigjson"]
			if !hasConfig {
				logger.Info("could not find any docker config in image pull secret")
			}

			type dockerConfig struct {
				Auths map[string]struct {
					Username string
					Password string
				}
			}

			var dockerConf dockerConfig
			err = json.Unmarshal(config, &dockerConf)
			if err != nil {
				logger.Error(err, err.Error())
			}

			//
			//logger.Error(err, err.Error())
			//logger.Info("image pull secret", "secret", imagePullSecret, "server", server, "digest", digest)

			server := strings.Split(status.Image, "/")[0]
			isLatest, err := isImageLatest(logger, instance, status, dockerConf.Auths["https://"+server].Username, dockerConf.Auths["https://"+server].Password)
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

func isImageLatest(logger logr.Logger, instance *dynatracev1alpha1.ActiveGate, status corev1.ContainerStatus, username string, password string) (bool, error) {
	server := strings.Split(status.Image, "/")[0]
	digest := strings.Split(status.ImageID, "@")[1]

	registry := docker.Registry{
		Server:   server,
		Image:    instance.Spec.Image,
		Username: username,
		Password: password,
	}

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
