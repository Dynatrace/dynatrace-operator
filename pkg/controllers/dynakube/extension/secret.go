package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	if !r.dk.PrometheusEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), extensionsTokenSecretConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), extensionsTokenSecretConditionType)

		secret, err := r.buildSecret(dttoken.Token{})
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
	if err != nil && !errors.IsNotFound(err) {
		log.Info("failed to check existence of extension secret")
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsTokenSecretConditionType, err)

		return err
	}

	if errors.IsNotFound(err) {
		newEecToken, err := dttoken.New(eecTokenSecretValuePrefix)
		if err != nil {
			log.Info("failed to generate eec token")
			conditions.SetSecretGenFailed(r.dk.Conditions(), extensionsTokenSecretConditionType, err)

			return err
		}

		newSecret, err := r.buildSecret(*newEecToken)
		if err != nil {
			log.Info("failed to generate extension secret")
			conditions.SetSecretGenFailed(r.dk.Conditions(), extensionsTokenSecretConditionType, err)

			return err
		}

		_, err = k8ssecret.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newSecret)
		if err != nil {
			log.Info("failed to create/update extension secret")
			conditions.SetKubeApiError(r.dk.Conditions(), extensionsTokenSecretConditionType, err)

			return err
		}
	}

	conditions.SetSecretCreated(r.dk.Conditions(), extensionsTokenSecretConditionType, r.getSecretName())

	return nil
}

func (r *reconciler) buildSecret(token dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecTokenSecretKey: []byte(token.String()),
	}

	return k8ssecret.Build(r.dk, r.getSecretName(), secretData)
}

func (r *reconciler) getSecretName() string {
	return r.dk.Name + secretSuffix
}
