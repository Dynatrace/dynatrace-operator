package ingestendpoint

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	agcapability "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/capability"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MetricsUrlSecretField   = "DT_METRICS_INGEST_URL"
	MetricsTokenSecretField = "DT_METRICS_INGEST_API_TOKEN"
	StatsdUrlSecretField    = "DT_STATSD_INGEST_URL"
	configFile              = "endpoint.properties"
)

// EndpointSecretGenerator manages the mint endpoint secret generation for the user namespaces.
type EndpointSecretGenerator struct {
	client    client.Client
	apiReader client.Reader
	namespace string
}

func NewEndpointSecretGenerator(client client.Client, apiReader client.Reader, ns string) *EndpointSecretGenerator {
	return &EndpointSecretGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
	}
}

// GenerateForNamespace creates the data-ingest-endpoint secret for namespace while only having the name of the corresponding dynakube
// Used by the podInjection webhook in case the namespace lacks the secret.
func (g *EndpointSecretGenerator) GenerateForNamespace(ctx context.Context, dkName, targetNs string) (bool, error) {
	log.Info("reconciling data-ingest endpoint secret for", "namespace", targetNs)
	var dk dynatracev1beta1.DynaKube
	if err := g.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: g.namespace}, &dk); err != nil {
		return false, err
	}

	data, err := g.prepare(ctx, &dk)
	if err != nil {
		return false, err
	}

	coreLabels := kubeobjects.NewCoreLabels(dkName, kubeobjects.ActiveGateComponentLabel)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretEndpointName,
			Namespace: targetNs,
			Labels:    coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
	return kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, secret, log)
}

// GenerateForDynakube creates/updates the data-ingest-endpoint secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *EndpointSecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1beta1.DynaKube) (bool, error) {
	log.Info("reconciling data-ingest endpoint secret for", "dynakube", dk.Name)
	anyUpdated := false

	data, err := g.prepare(ctx, dk)
	if err != nil {
		return anyUpdated, err
	}
	coreLabels := kubeobjects.NewCoreLabels(dk.Name, kubeobjects.ActiveGateComponentLabel)
	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return anyUpdated, err
	}
	for _, targetNs := range nsList {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretEndpointName,
				Namespace: targetNs.Name,
				Labels:    coreLabels.BuildMatchLabels(),
			},
			Data: data,
			Type: corev1.SecretTypeOpaque,
		}
		if upd, err := kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, secret, log); err != nil {
			return upd, err
		} else if upd {
			anyUpdated = true
		}
	}
	log.Info("done updating data-ingest endpoint secrets")
	return anyUpdated, nil
}

func (g *EndpointSecretGenerator) RemoveEndpointSecrets(ctx context.Context, dk *dynatracev1beta1.DynaKube) error {
	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return err
	}
	for _, targetNs := range nsList {
		endpointSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretEndpointName,
				Namespace: targetNs.GetName(),
			},
		}
		if err := g.client.Delete(context.TODO(), endpointSecret); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (g *EndpointSecretGenerator) prepare(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	fields, err := g.PrepareFields(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	endpointPropertiesBuilder := strings.Builder{}

	if !dk.FeatureDisableMetadataEnrichment() {
		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", MetricsUrlSecretField, fields[MetricsUrlSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}
		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", MetricsTokenSecretField, fields[MetricsTokenSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	if dk.NeedsStatsd() {
		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", StatsdUrlSecretField, fields[StatsdUrlSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	data := map[string][]byte{
		configFile: bytes.NewBufferString(endpointPropertiesBuilder.String()).Bytes(),
	}
	return data, nil
}

func (g *EndpointSecretGenerator) PrepareFields(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string]string, error) {
	fields := make(map[string]string)

	var tokens corev1.Secret
	if err := g.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if !dk.FeatureDisableMetadataEnrichment() {
		if token, ok := tokens.Data[dtclient.DynatraceDataIngestToken]; ok {
			fields[MetricsTokenSecretField] = string(token)
		}

		if diUrl, err := dataIngestUrl(dk); err != nil {
			return nil, err
		} else {
			fields[MetricsUrlSecretField] = diUrl
		}
	}

	if dk.NeedsStatsd() {
		if statsdUrl, err := statsdIngestUrl(dk); err != nil {
			return nil, err
		} else {
			fields[StatsdUrlSecretField] = statsdUrl
		}
	}

	return fields, nil
}

func dataIngestUrl(dk *dynatracev1beta1.DynaKube) (string, error) {
	if dk.IsActiveGateMode(dynatracev1beta1.MetricsIngestCapability.DisplayName) {
		return metricsIngestUrlForClusterActiveGate(dk)
	} else if len(dk.Spec.APIURL) > 0 {
		return metricsIngestUrlForDynatraceActiveGate(dk)
	} else {
		return "", fmt.Errorf("failed to create data-ingest endpoint, DynaKube.spec.apiUrl is empty")
	}
}

func metricsIngestUrlForDynatraceActiveGate(dk *dynatracev1beta1.DynaKube) (string, error) {
	return fmt.Sprintf("%s/v2/metrics/ingest", dk.Spec.APIURL), nil
}

func metricsIngestUrlForClusterActiveGate(dk *dynatracev1beta1.DynaKube) (string, error) {
	tenant, err := dk.TenantUUID()
	if err != nil {
		return "", err
	}

	serviceName := capability.BuildServiceName(dk.Name, agcapability.MultiActiveGateName)
	return fmt.Sprintf("https://%s.%s/e/%s/api/v2/metrics/ingest", serviceName, dk.Namespace, tenant), nil
}

func statsdIngestUrl(dk *dynatracev1beta1.DynaKube) (string, error) {
	serviceName := capability.BuildServiceName(dk.Name, agcapability.MultiActiveGateName)
	return fmt.Sprintf("%s.%s:%d", serviceName, dk.Namespace, agcapability.StatsdIngestPort), nil
}
