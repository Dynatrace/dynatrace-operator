package dtpullsecret

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	PullSecretSuffix = "-pull-secret"
)

type Reconciler struct {
	client.Client
	apiReader client.Reader
	instance  *v1alpha1.DynaKube
	dtc       dtclient.Client
	log       logr.Logger
	token     *v1.Secret
	image     string
	scheme    *runtime.Scheme
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *v1alpha1.DynaKube, dtc dtclient.Client, log logr.Logger, token *v1.Secret, image string) *Reconciler {
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

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	if r.isReconcileable(r.instance) {
		err := r.reconcilePullSecret()
		if err != nil {
			return activegate.LogError(r.log, err, "could not reconcile pull secret")
		}
	}

	return reconcile.Result{}, nil
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

	if err := controllerutil.SetControllerReference(r.instance, pullSecret, r.scheme); err != nil {
		r.log.Error(err, "error setting controller reference")
		return err
	}

	return r.updatePullSecretIfOutdated(pullSecret, pullSecretData)
}

func (r *Reconciler) createPullSecretIfNotExists(pullSecretData map[string][]byte) (*v1.Secret, error) {
	var config *v1.Secret
	err := r.apiReader.Get(context.TODO(), client.ObjectKey{Name: extendWithPullSecretSuffix(r.instance.Name), Namespace: r.instance.Namespace}, config)
	if k8serrors.IsNotFound(err) {
		r.log.Info("Creating ActiveGate config secret")
		return r.createPullSecret(pullSecretData)
	}
	return config, err
}

func (r *Reconciler) updatePullSecretIfOutdated(pullSecret *v1.Secret, desiredPullSecretData map[string][]byte) error {
	if !isPullSecretEqual(pullSecret, desiredPullSecretData) {
		return r.updatePullSecret(pullSecret, desiredPullSecretData)
	}
	return nil
}

func (r *Reconciler) createPullSecret(pullSecretData map[string][]byte) (*v1.Secret, error) {
	pullSecret := buildPullSecret(r.instance, pullSecretData)
	err := r.Create(context.TODO(), pullSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret '%s': %w", extendWithPullSecretSuffix(r.instance.Name), err)
	}
	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(pullSecret *v1.Secret, desiredPullSecretData map[string][]byte) error {
	r.log.Info(fmt.Sprintf("Updating secret %s", pullSecret.Name))
	pullSecret.Data = desiredPullSecretData
	if err := r.Update(context.TODO(), pullSecret); err != nil {
		return fmt.Errorf("failed to update secret %s: %w", pullSecret.Name, err)
	}
	return nil
}

func isPullSecretEqual(currentSecret *v1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func buildPullSecret(instance *v1alpha1.DynaKube, pullSecretData map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      extendWithPullSecretSuffix(instance.Name),
			Namespace: instance.Namespace,
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: pullSecretData,
	}
}

func extendWithPullSecretSuffix(name string) string {
	return name + PullSecretSuffix
}

func (r *Reconciler) isReconcileable(instance *v1alpha1.DynaKube) bool {
	return !r.hasCustomPullSecret(instance) && !r.hasImage()
}

func (r *Reconciler) hasCustomPullSecret(instance *v1alpha1.DynaKube) bool {
	return instance.Spec.CustomPullSecret != ""
}

func (r *Reconciler) hasImage() bool {
	return r.image != ""
}
