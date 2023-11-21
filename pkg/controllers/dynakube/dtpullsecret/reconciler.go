package dtpullsecret

import (
	"context"
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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
	client    client.Client
	apiReader client.Reader
	dynakube  *dynatracev1beta1.DynaKube
	scheme    *runtime.Scheme
	tokens    token.Tokens
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, tokens token.Tokens) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dynakube:  dynakube,
		tokens:    tokens,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dynakube.Spec.CustomPullSecret == "" {
		err := r.reconcilePullSecret(ctx)
		if err != nil {
			log.Info("could not reconcile pull secret")
			return errors.WithStack(err)
		}
	}

	return nil
}

func (r *Reconciler) reconcilePullSecret(ctx context.Context) error {
	pullSecretData, err := r.GenerateData()
	if err != nil {
		return errors.WithMessage(err, "could not generate pull secret data")
	}

	pullSecret, err := r.createPullSecretIfNotExists(ctx, pullSecretData)
	if err != nil {
		return errors.WithMessage(err, "failed to create or update secret")
	}

	return r.updatePullSecretIfOutdated(ctx, pullSecret, pullSecretData)
}

func (r *Reconciler) createPullSecretIfNotExists(ctx context.Context, pullSecretData map[string][]byte) (*corev1.Secret, error) {
	var config corev1.Secret
	err := r.apiReader.Get(ctx, client.ObjectKey{Name: extendWithPullSecretSuffix(r.dynakube.Name), Namespace: r.dynakube.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating pull secret")
		return r.createPullSecret(ctx, pullSecretData)
	}
	return &config, err
}

func (r *Reconciler) updatePullSecretIfOutdated(ctx context.Context, pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	if !isPullSecretEqual(pullSecret, desiredPullSecretData) {
		return r.updatePullSecret(ctx, pullSecret, desiredPullSecretData)
	}
	return nil
}

func (r *Reconciler) createPullSecret(ctx context.Context, pullSecretData map[string][]byte) (*corev1.Secret, error) {
	pullSecret, err := secret.Create(r.scheme, r.dynakube,
		secret.NewNameModifier(extendWithPullSecretSuffix(r.dynakube.Name)),
		secret.NewNamespaceModifier(r.dynakube.Namespace),
		secret.NewTypeModifier(corev1.SecretTypeDockerConfigJson),
		secret.NewDataModifier(pullSecretData))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = r.client.Create(ctx, pullSecret)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create secret %s", extendWithPullSecretSuffix(r.dynakube.Name))
	}
	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(ctx context.Context, pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	log.Info("updating secret", "name", pullSecret.Name)
	pullSecret.Data = desiredPullSecretData
	if err := r.client.Update(ctx, pullSecret); err != nil {
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
