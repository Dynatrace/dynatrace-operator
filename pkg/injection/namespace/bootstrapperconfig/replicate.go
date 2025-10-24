package bootstrapperconfig

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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

func (s *SecretGenerator) createSourceForWebhook(ctx context.Context, dk *dynakube.DynaKube, secretName, conditionType string, data map[string][]byte) error {
	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(secretName, dk.Namespace, data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	_, err = s.secrets.WithOwner(dk).CreateOrUpdate(ctx, secret)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	return nil
}
