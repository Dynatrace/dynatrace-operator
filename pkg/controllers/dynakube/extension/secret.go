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

const (
	eecTokenKey         = "eec-token"
	eecTokenValuePrefix = "EEC dt0x01"
	secretSuffix        = "-extensions-token"
)

func reconcileSecret(ctx context.Context, dk *dynakube.DynaKube, kubeClient client.Client, apiReader client.Reader) error {
	log.Info("reconciling secret")

	query := k8ssecret.NewQuery(ctx, kubeClient, apiReader, log)

	_, err := query.Get(client.ObjectKey{Name: getSecretName(dk.Name), Namespace: dk.Namespace})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		log.Info("creating secret")

		newEecToken, err := newEecToken()
		if err != nil {
			return err
		}

		newSecret, err := buildSecret(dk, *newEecToken)
		if err != nil {
			return err
		}

		query.CreateOrUpdate(*newSecret)
	}

	return nil
}

func buildSecret(dk *dynakube.DynaKube, token dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecTokenKey: []byte(token.String()),
	}

	return k8ssecret.Create(dk, k8ssecret.NewNameModifier(getSecretName(dk.Name)), k8ssecret.NewNamespaceModifier(dk.GetNamespace()), k8ssecret.NewDataModifier(secretData))
}

func newEecToken() (*dttoken.Token, error) {
	return dttoken.New(eecTokenValuePrefix)
}

func getSecretName(dynakubeName string) string {
	return dynakubeName + secretSuffix
}
