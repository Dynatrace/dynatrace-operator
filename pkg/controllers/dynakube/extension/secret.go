package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Remove in future release when migration is no longer needed
const DeprecatedOtelcTokenSecretKey = "otelc.token"

func (r *Reconciler) reconcileSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.Extensions().IsAnyEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), secretConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(dk.Conditions(), secretConditionType)

		secret, err := r.buildSecret(dttoken.Token{}, dttoken.Token{}, dk)
		if err != nil {
			log.Error(err, "failed to generate extension secret during cleanup")

			return nil
		}

		err = r.secrets.Delete(ctx, secret)
		if err != nil {
			log.Error(err, "failed to clean up extension secret")

			return nil
		}

		return nil
	}

	existingSecret, err := r.secrets.Get(ctx, client.ObjectKey{Name: r.getSecretName(dk), Namespace: dk.Namespace})
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info("failed to check existence of extension secret")
		k8sconditions.SetKubeAPIError(dk.Conditions(), secretConditionType, err)

		return err
	}

	// TODO: Remove in future release when migration is no longer needed
	migrationNeeded := r.removeDeprecatedSecretAndConditionIfNeeded(ctx, existingSecret, dk)

	if k8serrors.IsNotFound(err) || migrationNeeded {
		newEecToken, err := dttoken.New(eecConsts.TokenSecretValuePrefix)
		if err != nil {
			log.Info("failed to generate eec token")
			k8sconditions.SetSecretGenFailed(dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating eec token"))

			return err
		}

		newOtelcToken, err := dttoken.New(consts.DatasourceTokenSecretValuePrefix)
		if err != nil {
			log.Info("failed to generate otelc token")
			k8sconditions.SetSecretGenFailed(dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating otelc token"))

			return err
		}

		newSecret, err := r.buildSecret(*newEecToken, *newOtelcToken, dk)
		if err != nil {
			log.Info("failed to generate extension secret")
			k8sconditions.SetSecretGenFailed(dk.Conditions(), secretConditionType, err)

			return err
		}

		_, err = r.secrets.CreateOrUpdate(ctx, newSecret)
		if err != nil {
			log.Info("failed to create/update extension secret")
			k8sconditions.SetKubeAPIError(dk.Conditions(), secretConditionType, err)

			return err
		}
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), secretConditionType, r.getSecretName(dk))

	return nil
}

func (r *Reconciler) buildSecret(eecToken dttoken.Token, otelcToken dttoken.Token, dk *dynakube.DynaKube) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecConsts.TokenSecretKey:        []byte(eecToken.String()),
		consts.DatasourceTokenSecretKey: []byte(otelcToken.String()),
	}

	return k8ssecret.Build(dk, r.getSecretName(dk), secretData)
}

func (r *Reconciler) getSecretName(dk *dynakube.DynaKube) string {
	return dk.Extensions().GetTokenSecretName()
}

func (r *Reconciler) removeDeprecatedSecretAndConditionIfNeeded(ctx context.Context, existingSecret *corev1.Secret, dk *dynakube.DynaKube) bool {
	if existingSecret == nil {
		return false
	}

	if _, exists := existingSecret.Data[DeprecatedOtelcTokenSecretKey]; !exists {
		return false
	}

	if meta.FindStatusCondition(*dk.Conditions(), secretConditionType) == nil {
		return false
	}
	defer meta.RemoveStatusCondition(dk.Conditions(), secretConditionType)

	err := r.secrets.Delete(ctx, existingSecret)
	if err != nil {
		log.Error(err, "failed to delete old extension secret during migration")
	}

	return true
}
