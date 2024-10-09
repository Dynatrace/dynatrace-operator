package dtpullsecret

import (
	"context"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PullSecretSuffix = "-pull-secret"
)

type Reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dk           *dynakube.DynaKube
	tokens       token.Tokens
	timeprovider *timeprovider.Provider
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube, tokens token.Tokens) *Reconciler {
	return &Reconciler{
		client:       clt,
		apiReader:    apiReader,
		dk:           dk,
		tokens:       tokens,
		timeprovider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !(r.dk.NeedsOneAgent() || r.dk.ActiveGate().IsEnabled()) {
		if meta.FindStatusCondition(*r.dk.Conditions(), PullSecretConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		err := r.deletePullSecretIfExists(ctx)
		if err != nil {
			log.Error(err, "failed to clean-up pull secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), PullSecretConditionType)

		return nil
	}

	if conditions.IsOutdated(r.timeprovider, r.dk, PullSecretConditionType) {
		conditions.SetSecretOutdated(r.dk.Conditions(), PullSecretConditionType,
			extendWithPullSecretSuffix(r.dk.Name)+" is not present or outdated")

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

	err := r.apiReader.Get(ctx, client.ObjectKey{Name: extendWithPullSecretSuffix(r.dk.Name), Namespace: r.dk.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating pull secret")

		return r.createPullSecret(ctx, pullSecretData)
	}

	return &config, err
}

func (r *Reconciler) deletePullSecretIfExists(ctx context.Context) error {
	var config corev1.Secret

	err := r.apiReader.Get(ctx, client.ObjectKey{Name: extendWithPullSecretSuffix(r.dk.Name), Namespace: r.dk.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		return nil
	}

	log.Info("deleting pull secret")

	err = r.client.Delete(ctx, &config)
	if err != nil {
		return errors.WithMessage(err, "failed to delete pull secret")
	}

	return nil
}

func (r *Reconciler) updatePullSecretIfOutdated(ctx context.Context, pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	if !isPullSecretEqual(pullSecret, desiredPullSecretData) {
		return r.updatePullSecret(ctx, pullSecret, desiredPullSecretData)
	}

	return nil
}

func (r *Reconciler) createPullSecret(ctx context.Context, pullSecretData map[string][]byte) (*corev1.Secret, error) {
	pullSecret, err := secret.Build(r.dk,
		extendWithPullSecretSuffix(r.dk.Name),
		pullSecretData,
		secret.SetType(corev1.SecretTypeDockerConfigJson))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = r.client.Create(ctx, pullSecret)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), PullSecretConditionType, err)

		return nil, errors.WithMessagef(err, "failed to create secret %s", extendWithPullSecretSuffix(r.dk.Name))
	}

	conditions.SetSecretCreated(r.dk.Conditions(), PullSecretConditionType, pullSecret.Name)

	return pullSecret, nil
}

func (r *Reconciler) updatePullSecret(ctx context.Context, pullSecret *corev1.Secret, desiredPullSecretData map[string][]byte) error {
	log.Info("updating secret", "name", pullSecret.Name)

	pullSecret.Data = desiredPullSecretData
	if err := r.client.Update(ctx, pullSecret); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), PullSecretConditionType, err)

		return errors.WithMessagef(err, "failed to update secret %s", pullSecret.Name)
	}

	conditions.SetSecretUpdated(r.dk.Conditions(), PullSecretConditionType, pullSecret.Name)

	return nil
}

func isPullSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}

func extendWithPullSecretSuffix(name string) string {
	return name + PullSecretSuffix
}
