package dtpullsecret

import (
	"context"
	"reflect"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PullSecretSuffix = "-pull-secret"
)

type Reconciler struct {
	ctx       context.Context
	client    client.Client
	apiReader client.Reader
	dynakube  *dynatracev1.DynaKube
	scheme    *runtime.Scheme
	tokens    token.Tokens
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1.DynaKube, tokens token.Tokens) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		ctx:       ctx,
		client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dynakube:  dynakube,
		tokens:    tokens,
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
	err := r.apiReader.Get(r.ctx, client.ObjectKey{Name: extendWithPullSecretSuffix(r.dynakube.Name), Namespace: r.dynakube.Namespace}, &config)
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
	pullSecret, err := kubeobjects.CreateSecret(r.scheme, r.dynakube,
		kubeobjects.NewSecretNameModifier(extendWithPullSecretSuffix(r.dynakube.Name)),
		kubeobjects.NewSecretNamespaceModifier(r.dynakube.Namespace),
		kubeobjects.NewSecretTypeModifier(corev1.SecretTypeDockerConfigJson),
		kubeobjects.NewSecretDataModifier(pullSecretData))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = r.client.Create(r.ctx, pullSecret)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create secret %s", extendWithPullSecretSuffix(r.dynakube.Name))
	}
	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	log.Info("updating secret", "name", pullSecret.Name)
	pullSecret.Data = desiredPullSecretData
	if err := r.client.Update(r.ctx, pullSecret); err != nil {
		return errors.WithMessagef(err, "failed to update secret %s", pullSecret.Name)
	}
	return nil
}

func isPullSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func extendWithPullSecretSuffix(name string) string {
	return name + PullSecretSuffix
}
