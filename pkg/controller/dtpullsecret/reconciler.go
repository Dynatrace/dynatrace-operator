package dtpullsecret

import (
	"context"
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PullSecretSuffix = "-pull-secret"
)

type Reconciler struct {
	client.Client
	apiReader client.Reader
	instance  *dynatracev1alpha1.DynaKube
	dtc       dtclient.Client
	log       logr.Logger
	token     *corev1.Secret
	image     string
	scheme    *runtime.Scheme
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1alpha1.DynaKube, dtc dtclient.Client, log logr.Logger, token *corev1.Secret, image string) *Reconciler {
	return &Reconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		dtc:       dtc,
		log:       log,
		token:     token,
		image:     image,
	}
}

func (r *Reconciler) Reconcile() error {
	if !r.hasCustomPullSecret() && !r.hasImage() {
		err := r.reconcilePullSecret()
		if err != nil {
			r.log.Error(err, "could not reconcile pull secret")
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcilePullSecret() error {
	pullSecretData, err := r.GenerateData()
	if err != nil {
		return fmt.Errorf("could not generate pull secret data: %w", err)
	}

	pullSecret, err := r.createPullSecretIfNotExists(pullSecretData)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}

	return r.updatePullSecretIfOutdated(pullSecret, pullSecretData)
}

func (r *Reconciler) createPullSecretIfNotExists(pullSecretData map[string][]byte) (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(context.TODO(), client.ObjectKey{Name: extendWithPullSecretSuffix(r.instance.Name), Namespace: r.instance.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		r.log.Info("Creating pull secret")
		return r.createPullSecret(pullSecretData)
	}
	return &config, err
}

func (r *Reconciler) updatePullSecretIfOutdated(pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	if !isPullSecretEqual(pullSecret, desiredPullSecretData) {
		return r.updatePullSecret(pullSecret, desiredPullSecretData)
	}
	return nil
}

func (r *Reconciler) createPullSecret(pullSecretData map[string][]byte) (*corev1.Secret, error) {
	pullSecret := buildPullSecret(r.instance, pullSecretData)

	if err := controllerutil.SetControllerReference(r.instance, pullSecret, r.scheme); err != nil {
		return nil, err
	}

	err := r.Create(context.TODO(), pullSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret '%s': %w", extendWithPullSecretSuffix(r.instance.Name), err)
	}
	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	r.log.Info(fmt.Sprintf("Updating secret %s", pullSecret.Name))
	pullSecret.Data = desiredPullSecretData
	if err := r.Update(context.TODO(), pullSecret); err != nil {
		return fmt.Errorf("failed to update secret %s: %w", pullSecret.Name, err)
	}
	return nil
}

func isPullSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func buildPullSecret(instance *dynatracev1alpha1.DynaKube, pullSecretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      extendWithPullSecretSuffix(instance.Name),
			Namespace: instance.Namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: pullSecretData,
	}
}

func extendWithPullSecretSuffix(name string) string {
	return name + PullSecretSuffix
}

func (r *Reconciler) hasCustomPullSecret() bool {
	return r.instance.Spec.CustomPullSecret != ""
}

func (r *Reconciler) hasImage() bool {
	return r.image != ""
}
