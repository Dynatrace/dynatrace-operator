package activegate

import (
	"context"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	"regexp"
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
			if status.Image == "" {
				// If image is not yet pulled skip check
				continue
			}
			logger.Info("pods container status", "pod", pod.Name, "container", status.Name, "image id", status.ImageID)

			imagePullSecret := &corev1.Secret{}
			err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: pod.Namespace, Name: ImagePullSecret}, imagePullSecret)
			if err != nil {
				logger.Error(err, err.Error())
			}

			isLatest, err := isImageLatest(logger, &status, imagePullSecret)
			if err != nil {
				logger.Error(err, err.Error())
				//Error during image check, do nothing an continue with next status
				continue
			}

			if !isLatest {
				outdatedPods = append(outdatedPods, pod)
				// Pod is outdated, break loop
				break
			}
		}
	}

	return outdatedPods, nil
}

func isImageLatest(logger logr.Logger, status *corev1.ContainerStatus, imagePullSecret *corev1.Secret) (bool, error) {
	regex := regexp.MustCompile("(^docker-pullable:\\/\\/|\\:.*$|\\@sha256.*$)")
	latestImageName := regex.ReplaceAllString(status.Image, "") + ":latest"
	//Using ImageID instead of Image because ImageID contains digest of image that is used while Image only contains tag
	reference, err := name.ParseReference(strings.TrimPrefix(status.ImageID, "docker-pullable://"))
	if err != nil {
		return false, err
	}

	latestReference, err := name.ParseReference(latestImageName)
	if err != nil {
		return false, err
	}

	registryURL := "https://" + reference.Context().RegistryStr()
	authOption, err := getAuthOption(imagePullSecret, registryURL)
	if err != nil {
		logger.Info(err.Error())
	}

	latestDigest, err := getDigest(latestReference, authOption)
	if err != nil {
		return false, err
	}

	currentDigest, err := getDigest(reference, authOption)
	if err != nil {
		return false, err
	}

	logger.Info("Checked image against :latest",
		"latest digest", latestDigest,
		"currentDigest", currentDigest,
		"is latest", currentDigest == latestDigest)
	return currentDigest == latestDigest, nil
}

func getDigest(reference name.Reference, authOption remote.Option) (string, error) {
	img, err := remote.Image(reference, authOption)
	if err != nil {
		return "", err
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	return digest.Hex, nil
}

func getAuthOption(imagePullSecret *corev1.Secret, registryURL string) (remote.Option, error) {
	dockerConf, err := parser.NewDockerConfig(imagePullSecret)
	if err != nil {
		return remote.WithAuthFromKeychain(authn.DefaultKeychain), err
	} else {
		return remote.WithAuth(authn.FromConfig(authn.AuthConfig{
			Username: dockerConf.Auths[registryURL].Username,
			Password: dockerConf.Auths[registryURL].Password,
		})), nil
	}
}

func (r *ReconcileActiveGate) findPods(instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(builder.BuildLabelsForQuery(instance.Name)),
	}
	err := r.client.List(context.TODO(), podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

const (
	ImagePullSecret = "dynatrace-activegate-registry"
)
