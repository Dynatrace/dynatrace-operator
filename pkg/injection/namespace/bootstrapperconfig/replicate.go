package bootstrapperconfig

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	corev1 "k8s.io/api/core/v1"
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

func (s *SecretGenerator) createSourceForWebhook(ctx context.Context, dk *dynakube.DynaKube, secretName, conditionType string, data map[string][]byte, annotations map[string]string) error { //nolint:revive
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.WebhookComponentLabel)

	opts := []k8ssecret.Option{k8ssecret.SetLabels(coreLabels.BuildLabels())}
	if _, hasPullSecret := data[corev1.DockerConfigJsonKey]; hasPullSecret {
		opts = append(opts, k8ssecret.SetType(corev1.SecretTypeDockerConfigJson))
	}

	secret, err := k8ssecret.BuildForNamespace(secretName, dk.Namespace, data, opts...)
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	secret.Annotations = annotations

	_, err = s.secrets.WithOwner(dk).CreateOrUpdate(ctx, secret)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	return nil
}
