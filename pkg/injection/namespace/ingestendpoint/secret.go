package ingestendpoint

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MetricsUrlSecretField   = "DT_METRICS_INGEST_URL"
	MetricsTokenSecretField = "DT_METRICS_INGEST_API_TOKEN"
	configFile              = "endpoint.properties"
)

// SecretGenerator manages the mint endpoint secret generation for the user namespaces.
type SecretGenerator struct {
	client    client.Client
	apiReader client.Reader
	namespace string
}

func NewSecretGenerator(client client.Client, apiReader client.Reader, ns string) *SecretGenerator {
	return &SecretGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
	}
}

// GenerateForNamespace creates the metadata-enrichment-endpoint secret for namespace while only having the name of the corresponding dynakube
// Used by the podInjection webhook in case the namespace lacks the secret.
func (g *SecretGenerator) GenerateForNamespace(ctx context.Context, dkName, targetNs string) error {
	log.Info("reconciling metadata-enrichment endpoint secret for", "namespace", targetNs)

	var dk dynakube.DynaKube
	if err := g.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: g.namespace}, &dk); err != nil {
		return errors.WithStack(err)
	}

	data, err := g.prepare(ctx, &dk)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := labels.NewCoreLabels(dkName, labels.ActiveGateComponentLabel)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.EnrichmentEndpointSecretName,
			Namespace: targetNs,
			Labels:    coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
	secretQuery := k8ssecret.Query(g.client, g.apiReader, log)

	_, err = secretQuery.CreateOrUpdate(ctx, secret)

	return errors.WithStack(err)
}

// GenerateForDynakube creates/updates the metadata-enrichment-endpoint secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *SecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling metadata-enrichment endpoint secret for", "dynakube", dk.Name)

	data, err := g.prepare(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := labels.NewCoreLabels(dk.Name, labels.ActiveGateComponentLabel)

	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return errors.WithStack(err)
	}

	secretQuery := k8ssecret.Query(g.client, g.apiReader, log)
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:   consts.EnrichmentEndpointSecretName,
			Labels: coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	err = secretQuery.CreateOrUpdateForNamespaces(ctx, &secret, nsList)
	if err != nil {
		return err
	}

	log.Info("done updating metadata-enrichment endpoint secrets")

	return nil
}

func (g *SecretGenerator) RemoveEndpointSecrets(ctx context.Context, namespaces []corev1.Namespace) error {
	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	secretQuery := k8ssecret.Query(g.client, g.apiReader, log)

	return secretQuery.DeleteForNamespaces(ctx, consts.EnrichmentEndpointSecretName, nsList)
}

func (g *SecretGenerator) prepare(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	fields, err := g.PrepareFields(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	endpointPropertiesBuilder := strings.Builder{}

	if dk.MetadataEnrichmentEnabled() { // TODO: why check here and not at the very beginning?
		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", MetricsUrlSecretField, fields[MetricsUrlSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}

		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", MetricsTokenSecretField, fields[MetricsTokenSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	data := map[string][]byte{
		configFile: bytes.NewBufferString(endpointPropertiesBuilder.String()).Bytes(),
	}

	return data, nil
}

func (g *SecretGenerator) PrepareFields(ctx context.Context, dk *dynakube.DynaKube) (map[string]string, error) {
	fields := make(map[string]string)

	tokens, err := k8ssecret.Query(g.client, g.apiReader, log).Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: g.namespace})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if dk.MetadataEnrichmentEnabled() { // TODO: why check here and not at the very beginning?
		if token, ok := tokens.Data[dtclient.DataIngestToken]; ok {
			fields[MetricsTokenSecretField] = string(token)
		}

		if ingestUrl, err := ingestUrlFor(dk); err != nil {
			return nil, err
		} else {
			fields[MetricsUrlSecretField] = ingestUrl
		}
	}

	return fields, nil
}

func ingestUrlFor(dk *dynakube.DynaKube) (string, error) {
	switch {
	case dk.ActiveGate().IsMetricsIngestEnabled():
		return metricsIngestUrlForClusterActiveGate(dk)
	case len(dk.Spec.APIURL) > 0:
		return metricsIngestUrlForDynatraceActiveGate(dk)
	default:
		return "", errors.New("failed to create metadata-enrichment endpoint, DynaKube.spec.apiUrl is empty")
	}
}

func metricsIngestUrlForDynatraceActiveGate(dk *dynakube.DynaKube) (string, error) {
	return dk.Spec.APIURL + "/v2/metrics/ingest", nil
}

func metricsIngestUrlForClusterActiveGate(dk *dynakube.DynaKube) (string, error) {
	tenant, err := dk.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return "", err
	}

	serviceName := capability.BuildServiceName(dk.Name, agconsts.MultiActiveGateName)

	return fmt.Sprintf("http://%s.%s/e/%s/api/v2/metrics/ingest", serviceName, dk.Namespace, tenant), nil
}
