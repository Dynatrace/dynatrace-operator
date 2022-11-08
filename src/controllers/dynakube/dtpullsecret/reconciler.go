package dtpullsecret

import (
	"context"
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PullSecretSuffix = "-pull-secret"
)

type Reconciler struct {
	client              client.Client
	apiReader           client.Reader
	dynakube            *dynatracev1beta1.DynaKube
	scheme              *runtime.Scheme
	apiToken, paasToken string
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, apiToken, paasToken string) *Reconciler {
	if paasToken == "" {
		paasToken = apiToken
	}
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dynakube:  dynakube,
		apiToken:  apiToken,
		paasToken: paasToken,
	}
}

func (r *Reconciler) Reconcile() error {
	if r.dynakube.Spec.CustomPullSecret == "" {
		err := r.reconcilePullSecret()
		if err != nil {
			log.Info("could not reconcile pull secret")
			return errors.WithStack(err)
		}
	}

	return nil
}

func (r *Reconciler) reconcilePullSecret() error {
	pullSecretData, err := r.GenerateData()
	if err != nil {
		return errors.WithMessage(err, "could not generate pull secret data")
	}

	pullSecret, err := r.createPullSecretIfNotExists(pullSecretData)
	if err != nil {
		return errors.WithMessage(err, "failed to create or update secret")
	}

	return r.updatePullSecretIfOutdated(pullSecret, pullSecretData)
}

func (r *Reconciler) createPullSecretIfNotExists(pullSecretData map[string][]byte) (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(context.TODO(), client.ObjectKey{Name: extendWithPullSecretSuffix(r.dynakube.Name), Namespace: r.dynakube.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating pull secret")
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
	pullSecret := BuildPullSecret(r.dynakube, pullSecretData)

	if err := controllerutil.SetControllerReference(r.dynakube, pullSecret, r.scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	err := r.client.Create(context.TODO(), pullSecret)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create secret %s", extendWithPullSecretSuffix(r.dynakube.Name))
	}
	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	log.Info("updating secret", "name", pullSecret.Name)
	pullSecret.Data = desiredPullSecretData
	if err := r.client.Update(context.TODO(), pullSecret); err != nil {
		return errors.WithMessagef(err, "failed to update secret %s", pullSecret.Name)
	}
	return nil
}

func isPullSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func BuildPullSecret(dynakube *dynatracev1beta1.DynaKube, pullSecretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      extendWithPullSecretSuffix(dynakube.Name),
			Namespace: dynakube.Namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: pullSecretData,
	}
}

func extendWithPullSecretSuffix(name string) string {
	return name + PullSecretSuffix
}
