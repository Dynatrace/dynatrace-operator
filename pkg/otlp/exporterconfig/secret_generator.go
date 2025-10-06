package exporterconfig

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretGenerator manages the OTLP exporter secret secret generation for the user namespaces.
type SecretGenerator struct {
	client       client.Client
	dtClient     dtclient.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewSecretGenerator(client client.Client, apiReader client.Reader, dtClient dtclient.Client) *SecretGenerator {
	return &SecretGenerator{
		client:       client,
		dtClient:     dtClient,
		apiReader:    apiReader,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(client, apiReader, log),
	}
}

// GenerateForDynakube creates/updates the OTLP exporter config secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (s *SecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling namespace bootstrapper init secret for", "dynakube", dk.Name)

	data, err := s.generateConfig(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	nsList, err := mapper.GetNamespacesForDynakube(ctx, s.apiReader, dk.Name)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return errors.WithStack(err)
	}

	if len(data) != 0 {
		err = s.createSourceForWebhook(ctx, dk, GetSourceConfigSecretName(dk.Name), ConfigConditionType, data)
		if err != nil {
			return err
		}

		err = s.createSecretForNSlist(ctx, consts.OTLPExporterSecretName, ConfigConditionType, nsList, dk, data)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func Cleanup(ctx context.Context, client client.Client, apiReader client.Reader, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	err := cleanupConfig(ctx, client, apiReader, namespaces, dk)
	if err != nil {
		log.Error(err, "failed to cleanup bootstrapper config secrets")

		return errors.WithStack(err)
	}

	return nil
}

func (s *SecretGenerator) generateConfig(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	data := map[string][]byte{}

	if dk.Spec.OTLPExporterConfiguration == nil {
		return data, nil
	}

	var tokens corev1.Secret
	if err := s.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tokens); err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if _, ok := tokens.Data[dtclient.DataIngestToken]; !ok {
		err := errors.New("data ingest token not found in tokens secret")
		conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	data[dtclient.DataIngestToken] = tokens.Data[dtclient.DataIngestToken]

	return data, nil
}

func (s *SecretGenerator) createSecretForNSlist( //nolint:revive // argument-limit
	ctx context.Context,
	secretName string,
	conditionType string,
	nsList []corev1.Namespace,
	dk *dynakube.DynaKube,
	data map[string][]byte,
) error {
	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(secretName, "", data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), conditionType, err)

		return err
	}

	err = s.secrets.CreateOrUpdateForNamespaces(ctx, secret, nsList)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	log.Info("done updating OTLP exporter secrets")
	conditions.SetSecretCreatedOrUpdated(dk.Conditions(), conditionType, consts.OTLPExporterSecretName)

	return nil
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

func cleanupConfig(ctx context.Context, client client.Client, apiReader client.Reader, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	defer meta.RemoveStatusCondition(dk.Conditions(), ConfigConditionType)

	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	secrets := k8ssecret.Query(client, apiReader, log)

	err := secrets.DeleteForNamespace(ctx, GetSourceConfigSecretName(dk.Name), dk.Namespace)
	if err != nil {
		log.Error(err, "failed to delete the source bootstrapper-config secret", "name", GetSourceConfigSecretName(dk.Name))
	}

	return secrets.DeleteForNamespaces(ctx, consts.OTLPExporterSecretName, nsList)
}
