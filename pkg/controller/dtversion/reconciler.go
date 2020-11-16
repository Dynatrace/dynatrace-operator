package dtversion

import (
	"context"
	"errors"
	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type Reconciler struct {
	client.Client
	scheme   *runtime.Scheme
	dtc      dtclient.Client
	log      logr.Logger
	token    *corev1.Secret
	instance *v1alpha1.DynaKube
}

func NewReconciler() *Reconciler {
	return &Reconciler{}
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	pods, err := r.findPods()
	if err != nil {
		return activegate.LogError(r.log, err, "could not list pods")
	}

	err = r.setVersionLabelForPods(pods)
	if err != nil {
		var statusError *k8serrors.StatusError
		if errors.As(err, &statusError) {
			// Since this happens early during deployment, pods might have been modified
			// In this case, retry silently
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
		}
		// Otherwise, retry loudly
		return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) findPods() ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	err := r.List(context.TODO(), podList, buildListOptions(r.instance)...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (r *Reconciler) setVersionLabelForPods(pods []corev1.Pod) error {
	for i := range pods {
		err := r.setVersionLabel(&pods[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) setVersionLabel(pod *corev1.Pod) error {
	versionLabel, err := r.getVersionLabelForPod(pod)
	if err != nil {
		return err
	}

	pod.Labels[VersionKey] = versionLabel
	err = r.Update(context.TODO(), pod)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) getVersionLabelForPod(pod *corev1.Pod) (string, error) {
	result := ""
	for _, status := range pod.Status.ContainerStatuses {
		if status.Image == "" {
			// If Image is not present, skip
			continue
		}

		imagePullSecret, err := dtpullsecret.GetImagePullSecret(r, r.instance)
		if err != nil {
			// Something wrong with pull secret, exit function entirely
			return "", err
		}

		dockerConfig, err := NewDockerConfig(imagePullSecret)
		// If an error is returned, try getting the image anyway

		versionLabel, err2 := GetVersionLabel(status.Image, dockerConfig)
		if err2 != nil && err != nil {
			// If an error is returned when getting labels and an error occurred during parsing of the docker config
			// assume the error from parsing the docker config is the reason
			return "", err
		} else if err2 != nil {
			return "", err2
		}

		if result == "" {
			result = versionLabel
		}
	}

	return result, nil
}

func buildListOptions(instance *v1alpha1.DynaKube) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(kubemon.BuildLabelsFromInstance(instance)),
	}
}
