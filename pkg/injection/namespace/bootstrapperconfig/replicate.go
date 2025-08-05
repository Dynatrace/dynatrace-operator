package bootstrapperconfig

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	sourceSecretTemplate      = "%s-bootstrapper-config"
	sourceSecretCertsTemplate = "%s-bootstrapper-certs"
)

func GetSourceConfigSecretName(dkName string) string {
	return fmt.Sprintf(sourceSecretTemplate, dkName)
}

func GetSourceCertsSecretName(dkName string) string {
	return fmt.Sprintf(sourceSecretCertsTemplate, dkName)
}

// Replicate will only create the secret once, doesn't mean for keeping the secret up to date
func Replicate(ctx context.Context, dk dynakube.DynaKube, query k8ssecret.QueryObject, sourceSecretName, targetSecretName string, targetNs string) error { //nolint:revive
	secret, err := getSecretFromSource(ctx, dk, query, sourceSecretName, targetSecretName, targetNs)
	if err != nil {
		return err
	}

	return client.IgnoreAlreadyExists(query.Create(ctx, secret))
}

func getSecretFromSource(ctx context.Context, dk dynakube.DynaKube, query k8ssecret.QueryObject, sourceSecretName, targetSecretName string, targetNs string) (*corev1.Secret, error) { //nolint:revive
	source, err := query.Get(ctx, types.NamespacedName{Name: sourceSecretName, Namespace: dk.Namespace})
	if err != nil {
		return nil, err
	}

	return k8ssecret.BuildForNamespace(targetSecretName, targetNs, source.Data, k8ssecret.SetLabels(source.Labels))
}

func (s *SecretGenerator) createSourceForWebhook(ctx context.Context, dk *dynakube.DynaKube, secretName, conditionType string, data map[string][]byte) error {
	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(secretName, dk.Namespace, data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	_, err = s.secretQuery.WithOwner(dk).CreateOrUpdate(ctx, secret)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	return nil
}
