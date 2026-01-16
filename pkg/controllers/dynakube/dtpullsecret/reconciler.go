package dtpullsecret

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
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
	dk      *dynakube.DynaKube
	tokens  token.Tokens
	secrets k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube, tokens token.Tokens) *Reconciler {
	return &Reconciler{
		dk:      dk,
		tokens:  tokens,
		secrets: k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.OneAgent().IsDaemonsetRequired() && !r.dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), PullSecretConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		defer meta.RemoveStatusCondition(r.dk.Conditions(), PullSecretConditionType)

		secret, _ := k8ssecret.Build(r.dk, extendWithPullSecretSuffix(r.dk.Name), nil)

		_ = r.deleteSecret(ctx, secret)

		return nil
	}

	// no DT API request is made here
	err := r.reconcilePullSecret(ctx)
	if err != nil {
		log.Info("could not reconcile pull secret")

		return errors.WithStack(err)
	}

	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	log.Info("deleting pull secret", "name", secret.Name)

	err := r.secrets.Delete(ctx, secret)
	if err != nil && !k8serrors.IsNotFound(err) {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), PullSecretConditionType, err)

		return errors.WithMessagef(err, "failed to delete secret %s", secret.Name)
	}

	return nil
}

func (r *Reconciler) reconcilePullSecret(ctx context.Context) error {
	pullSecretData, err := r.generateData()
	if err != nil {
		return errors.WithMessage(err, "could not generate pull secret data")
	}

	secret, err := k8ssecret.Build(r.dk,
		extendWithPullSecretSuffix(r.dk.Name), pullSecretData,
		k8ssecret.SetType(corev1.SecretTypeDockerConfigJson),
	)
	if err != nil {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), PullSecretConditionType, err)

		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret", "name", secret.Name)
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), PullSecretConditionType, errors.WithMessage(err, "failed to create or update secret"))

		return errors.WithMessage(err, "failed to create or update secret")
	}

	k8sconditions.SetSecretCreated(r.dk.Conditions(), PullSecretConditionType, secret.Name)

	return nil
}

func extendWithPullSecretSuffix(name string) string {
	return name + PullSecretSuffix
}
