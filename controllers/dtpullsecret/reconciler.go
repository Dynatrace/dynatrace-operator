package dtpullsecret

import (
	"context"
	"fmt"
	"reflect"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PullSecretSuffix = "-pull-secret"

	//possible metrics
	FailedCreatePullSecretEvent = "FailedCreatePullSecret"
	CreatePullSecretEvent       = "CreatePullSecret"
	FailedUpdatePullSecretEvent = "FailedUpdatePullSecret"
	UpdatePullSecretEvent       = "UpdatePullSecret"
)

type Reconciler struct {
	client.Client
	apiReader client.Reader
	instance  *dynatracev1alpha1.DynaKube
	log       logr.Logger
	token     *corev1.Secret
	scheme    *runtime.Scheme
	recorder  record.EventRecorder
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1alpha1.DynaKube, log logr.Logger, token *corev1.Secret, recorder record.EventRecorder) *Reconciler {
	return &Reconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		log:       log,
		token:     token,
		recorder:  recorder,
	}
}

func (r *Reconciler) Reconcile() error {
	if r.instance.Spec.CustomPullSecret == "" {
		err := r.reconcilePullSecret()
		if err != nil {
			r.log.Error(err, "could not reconcile pull secret")
			return errors.WithStack(err)
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
	pullSecret := BuildPullSecret(r.instance, pullSecretData)

	if err := controllerutil.SetControllerReference(r.instance, pullSecret, r.scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	err := r.Create(context.TODO(), pullSecret)
	if err != nil {
		err = fmt.Errorf("failed to create secret '%s': %w", extendWithPullSecretSuffix(r.instance.Name), err)
		r.recorder.Event(pullSecret,
			corev1.EventTypeWarning,
			FailedCreatePullSecretEvent,
			err.Error())
		return nil, err
	}
	r.recorder.Event(pullSecret,
		corev1.EventTypeNormal,
		CreatePullSecretEvent,
		"Created pull secret.")
	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	r.log.Info(fmt.Sprintf("Updating secret %s", pullSecret.Name))
	pullSecret.Data = desiredPullSecretData
	if err := r.Update(context.TODO(), pullSecret); err != nil {
		err = fmt.Errorf("failed to update secret %s: %w", pullSecret.Name, err)
		r.recorder.Event(pullSecret,
			corev1.EventTypeWarning,
			FailedUpdatePullSecretEvent,
			err.Error())
		return err
	}
	r.recorder.Event(pullSecret,
		corev1.EventTypeNormal,
		UpdatePullSecretEvent,
		"Updated pull secret.")
	return nil
}

func isPullSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func BuildPullSecret(instance *dynatracev1alpha1.DynaKube, pullSecretData map[string][]byte) *corev1.Secret {
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
