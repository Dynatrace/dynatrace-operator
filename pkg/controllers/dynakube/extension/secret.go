package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	log.Info("reconciling secret " + getSecretName(r.dk.Name))

	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

	if !r.dk.PrometheusEnabled() {
		err := query.Delete(getSecretName(r.dk.Name), r.dk.Namespace)
		if err != nil {
			return err
		}

		removeSecretCreatedCondition(r.dk.Conditions())

		return nil
	}

	_, err := query.Get(client.ObjectKey{Name: getSecretName(r.dk.Name), Namespace: r.dk.Namespace})
	if err != nil && !errors.IsNotFound(err) {
		setSecretCreatedFailureCondition(r.dk.Conditions(), err)

		return err
	}

	if errors.IsNotFound(err) {
		log.Info("creating secret " + getSecretName(r.dk.Name))

		newEecToken, err := dttoken.New(eecTokenSecretValuePrefix)
		if err != nil {
			setSecretCreatedFailureCondition(r.dk.Conditions(), err)

			return err
		}

		newSecret, err := buildSecret(r.dk, *newEecToken)
		if err != nil {
			setSecretCreatedFailureCondition(r.dk.Conditions(), err)

			return err
		}

		err = query.CreateOrUpdate(*newSecret)
		if err != nil {
			setSecretCreatedFailureCondition(r.dk.Conditions(), err)

			return err
		}
	}

	setSecretCreatedSuccessCondition(r.dk.Conditions())

	return nil
}

func buildSecret(dk *dynakube.DynaKube, token dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecTokenSecretKey: []byte(token.String()),
	}

	return k8ssecret.Create(dk, k8ssecret.NewNameModifier(getSecretName(dk.Name)), k8ssecret.NewNamespaceModifier(dk.GetNamespace()), k8ssecret.NewDataModifier(secretData))
}

func getSecretName(dynakubeName string) string {
	return dynakubeName + secretSuffix
}
