package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	if !r.dk.IsExtensionsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), secretConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), secretConditionType)

		secret, err := r.buildSecret(dttoken.Token{}, dttoken.Token{})
		if err != nil {
			log.Error(err, "failed to generate extension secret during cleanup")

			return nil
		}

		err = k8ssecret.Query(r.client, r.apiReader, log).Delete(ctx, secret)
		if err != nil {
			log.Error(err, "failed to clean up extension secret")

			return nil
		}

		return nil
	}

	_, err := k8ssecret.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKey{Name: r.getSecretName(), Namespace: r.dk.Namespace})
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info("failed to check existence of extension secret")
		conditions.SetKubeApiError(r.dk.Conditions(), secretConditionType, err)

		return err
	}

	if k8serrors.IsNotFound(err) {
		newEecToken, err := dttoken.New(eecConsts.TokenSecretValuePrefix)
		if err != nil {
			log.Info("failed to generate eec token")
			conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating eec token"))

			return err
		}

		newOtelcToken, err := dttoken.New(consts.OtelcTokenSecretValuePrefix)
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

		_, err = k8ssecret.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newSecret)
		if err != nil {
			log.Info("failed to create/update extension secret")
			conditions.SetKubeApiError(r.dk.Conditions(), secretConditionType, err)

			return err
		}
	}

	conditions.SetSecretCreated(r.dk.Conditions(), secretConditionType, r.getSecretName())

	return nil
}

func (r *reconciler) buildSecret(eecToken dttoken.Token, otelcToken dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecConsts.TokenSecretKey:   []byte(eecToken.String()),
		consts.OtelcTokenSecretKey: []byte(otelcToken.String()),
	}

	return k8ssecret.Build(r.dk, r.getSecretName(), secretData)
}

func (r *reconciler) getSecretName() string {
	return r.dk.ExtensionsTokenSecretName()
}
