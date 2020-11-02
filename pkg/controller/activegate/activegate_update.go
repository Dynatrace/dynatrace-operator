package activegate

import (
	"context"
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/version"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

/*
updateService provides an interface to update outdated pods.
The interface is used to increase testability of the Reconciler
Previously, the Reconciler was harder to unit test, because the methods of this interface depend on one another.
Additionally, the production code used makes api requests.
To allow mocking and testing of single methods used, this interface has been introduced.
WIth it, single methods can be overwritten or mocked to allow focused unti testing
*/
type updateService interface {
	FindOutdatedPods(
		r *ReconcileActiveGate,
		logger logr.Logger,
		instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error)
	IsLatest(validator version.ReleaseValidator) (bool, error)
	UpdatePods(
		r *ReconcileActiveGate,
		instance *dynatracev1alpha1.DynaKube) (*reconcile.Result, error)
}

/*
activeGateUpdateService provides the production implementation of an updateService.
Used by the Reconciler when the operator is running normally.
*/
type activeGateUpdateService struct{}

func (us *activeGateUpdateService) UpdatePods(
	r *ReconcileActiveGate,
	instance *dynatracev1alpha1.DynaKube) (*reconcile.Result, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	} else if !instance.Spec.KubernetesMonitoringSpec.DisableActivegateUpdate { // &&
		//instance.Status.UpdatedTimestamp.Add(UpdateInterval).Before(time.Now()) {
		log.Info("checking for outdated pods")
		// Check if pods have latest activegate version
		outdatedPods, err := r.updateService.FindOutdatedPods(r, log, instance)
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

		instance.Status.UpdatedTimestamp = metav1.Now()
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			log.Info("failed to updated instance status", "message", err.Error())
		}
	} else if instance.Spec.KubernetesMonitoringSpec.DisableActivegateUpdate {
		log.Info("Skipping updating pods because of configuration", "disableActivegateUpdate", true)
	}
	return nil, nil
}

func (us *activeGateUpdateService) FindOutdatedPods(
	r *ReconcileActiveGate,
	logger logr.Logger,
	instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error) {
	pods, err := r.findPods(instance)
	if err != nil {
		logger.Error(err, "failed to list pods")
		return nil, err
	}

	var outdatedPods []corev1.Pod
	for _, pod := range pods {
		for _, status := range pod.Status.ContainerStatuses {
			if status.ImageID == "" || status.Image == "" {
				// If image is not yet pulled or not given skip check
				continue
			}
			logger.Info("pods container status", "pod", pod.Name, "container", status.Name, "image id", status.ImageID)

			imagePullSecret, err := dao.GetImagePullSecret(r.client, &pod)
			if err != nil {
				logger.Error(err, err.Error())
			}

			dockerConfig, err := parser.NewDockerConfig(imagePullSecret)
			if err != nil {
				logger.Info(err.Error())
			}

			dockerHashesChecker := version.NewDockerLabelsChecker(instance.Spec.KubernetesMonitoringSpec.Image, pod.Labels, dockerConfig)

			isLatest, err := r.updateService.IsLatest(dockerHashesChecker)
			if err != nil {
				logger.Error(err, err.Error())
				//Error during image check, do nothing an continue with next status
				continue
			}

			if !isLatest {
				logger.Info("pod is outdated", "name", pod.Name)
				outdatedPods = append(outdatedPods, pod)
				// Pod is outdated, break loop
				break
			}
		}
	}

	return outdatedPods, nil
}

func (us *activeGateUpdateService) IsLatest(validator version.ReleaseValidator) (bool, error) {
	return validator.IsLatest()
}
