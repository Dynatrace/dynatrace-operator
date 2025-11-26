package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Remove in future release when migration is no longer needed
const DeprecatedOtelcTokenSecretKey = "otelc.token"

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	if !r.dk.Extensions().IsAnyEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), secretConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), secretConditionType)

		secret, err := r.buildSecret(dttoken.Token{}, dttoken.Token{})
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

	existingSecret, err := r.secrets.Get(ctx, client.ObjectKey{Name: r.getSecretName(), Namespace: r.dk.Namespace})
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info("failed to check existence of extension secret")
		conditions.SetKubeAPIError(r.dk.Conditions(), secretConditionType, err)

		return err
	}

	// TODO: Remove in future release when migration is no longer needed
	migrationNeeded := r.removeDeprecatedSecretAndConditionIfNeeded(ctx, existingSecret)

	if k8serrors.IsNotFound(err) || migrationNeeded {
		newEecToken, err := dttoken.New(eecConsts.TokenSecretValuePrefix)
		if err != nil {
			log.Info("failed to generate eec token")
			conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating eec token"))

			return err
		}

		newOtelcToken, err := dttoken.New(consts.DatasourceTokenSecretValuePrefix)
		if err != nil {
			log.Info("failed to generate otelc token")
			conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating otelc token"))

			return err
		}

		newSecret, err := r.buildSecret(*newEecToken, *newOtelcToken)
		if err != nil {
			log.Info("failed to generate extension secret")
			conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, err)

			return err
		}

		_, err = r.secrets.CreateOrUpdate(ctx, newSecret)
		if err != nil {
			log.Info("failed to create/update extension secret")
			conditions.SetKubeAPIError(r.dk.Conditions(), secretConditionType, err)

			return err
		}
	}

	conditions.SetSecretCreated(r.dk.Conditions(), secretConditionType, r.getSecretName())

	return nil
}

func (r *reconciler) buildSecret(eecToken dttoken.Token, otelcToken dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecConsts.TokenSecretKey:        []byte(eecToken.String()),
		consts.DatasourceTokenSecretKey: []byte(otelcToken.String()),
	}

	return k8ssecret.Build(r.dk, r.getSecretName(), secretData)
}

func (r *reconciler) getSecretName() string {
	return r.dk.Extensions().GetTokenSecretName()
}

func (r *reconciler) removeDeprecatedSecretAndConditionIfNeeded(ctx context.Context, existingSecret *corev1.Secret) bool {
	if existingSecret == nil {
		return false
	}

	if _, exists := existingSecret.Data[DeprecatedOtelcTokenSecretKey]; !exists {
		return false
	}

	if meta.FindStatusCondition(*r.dk.Conditions(), secretConditionType) == nil {
		return false
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), secretConditionType)

	err := r.secrets.Delete(ctx, existingSecret)
	if err != nil {
		log.Error(err, "failed to delete old extension secret during migration")
	}

	return true
}
