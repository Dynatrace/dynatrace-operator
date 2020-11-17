package dtpods

import (
	"context"
	"time"

	v1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtversion"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	UpdateInterval = 5 * time.Minute
)

type Reconciler struct {
	client.Client
	log         logr.Logger
	instance    *v1alpha1.DynaKube
	matchLabels map[string]string
	image       string
}

func NewReconciler(clt client.Client, log logr.Logger, instance *v1alpha1.DynaKube,
	matchLabels map[string]string, image string) *Reconciler {
	return &Reconciler{
		Client:      clt,
		log:         log,
		instance:    instance,
		matchLabels: matchLabels,
		image:       image,
	}
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	if isInstanceOutdated(r.instance) {
		err := r.updatePods()
		if err != nil {
			r.log.Error(err, err.Error())
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) updatePods() error {
	pods, err := NewPodFinder(r, r.instance, r.matchLabels).FindPods()
	if err != nil {
		return err
	}

	for _, pod := range pods {
		isOutdated, err := r.isPodOutdated(pod)
		if err != nil {
			r.log.Error(err, err.Error())
		}
		if isOutdated {
			err = r.deletePod(&pod)
			if err != nil {
				r.log.Error(err, err.Error())
			}
		}
	}

	return r.updateInstanceStatus()
}

func (r *Reconciler) isPodOutdated(pod corev1.Pod) (bool, error) {
	if _, hasVersionLabel := pod.Labels[dtversion.VersionKey]; !hasVersionLabel {
		r.log.Info("pod does not have '%s' label, skipping update check", dtversion.VersionKey, "pod", pod.Name)
		return false, nil
	}

	return r.hasOutdatedContainerStatus(pod)
}

func (r *Reconciler) hasOutdatedContainerStatus(pod corev1.Pod) (bool, error) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		isOutdated, err := r.isOutdatedContainerStatus(pod, containerStatus)
		if isOutdated {
			return isOutdated, err
		}
	}

	return false, nil
}

func (r *Reconciler) isOutdatedContainerStatus(pod corev1.Pod, status corev1.ContainerStatus) (bool, error) {
	if status.ImageID == "" || status.Image == "" {
		return false, nil
	}

	imagePullSecret, err := dtpullsecret.GetImagePullSecret(r, r.instance)
	if err != nil {
		// No return, try without pull secret
		r.log.Error(err, err.Error())
	}

	dockerConfig, err := dtversion.NewDockerConfig(imagePullSecret)
	if err != nil {
		// No return, try without docker config
		r.log.Error(err, err.Error())
	}

	isLatest, err := dtversion.NewDockerLabelsChecker(r.image, pod.Labels, dockerConfig).IsLatest()
	if err != nil {
		return false, err
	}
	return !isLatest, nil
}

func (r *Reconciler) deletePod(pod *corev1.Pod) error {
	return r.Delete(context.TODO(), pod)
}

func (r *Reconciler) updateInstanceStatus() error {
	r.instance.Status.UpdatedTimestamp = metav1.Now()
	err := r.Status().Update(context.TODO(), r.instance)
	if err != nil {
		r.log.Info("failed to updated instance status", "message", err.Error())
	}
	return err
}

func isInstanceOutdated(instance *v1alpha1.DynaKube) bool {
	//return !instance.Spec.KubernetesMonitoringSpec.DisableActivegateUpdate &&
	return instance.Status.UpdatedTimestamp.Add(UpdateInterval).Before(time.Now())
}
