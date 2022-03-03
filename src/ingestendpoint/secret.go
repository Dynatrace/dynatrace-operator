package ingestendpoint

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
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
	UrlSecretField   = "DT_METRICS_INGEST_URL"
	TokenSecretField = "DT_METRICS_INGEST_API_TOKEN"
	StatsdIngestUrl  = "DT_STATSD_INGEST_URL"
	configFile       = "endpoint.properties"
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
	return kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, SecretEndpointName, targetNs, data, corev1.SecretTypeOpaque, log)
}

// GenerateForDynakube creates/updates the data-ingest-endpoint secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *EndpointSecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1beta1.DynaKube) (bool, error) {
	log.Info("reconciling data-ingest endpoint secret for", "dynakube", dk.Name)

	data, err := g.prepare(ctx, dk)
	if err != nil {
		return false, err
	}

	anyUpdate := false
	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return false, err
	}
	for _, targetNs := range nsList {
		if upd, err := kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, SecretEndpointName, targetNs.Name, data, corev1.SecretTypeOpaque, log); err != nil {
			return upd, err
		} else if upd {
			anyUpdate = true
		}
	}
	log.Info("done updating data-ingest endpoint secrets")
	return anyUpdate, nil
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

	var endpointBuf bytes.Buffer

	//if dk.NeedsMetricsIngest() {
	if _, err := endpointBuf.WriteString(fmt.Sprintf("%s=%s\n", UrlSecretField, fields[UrlSecretField])); err != nil {
		return nil, errors.WithStack(err)
	}
	if _, err := endpointBuf.WriteString(fmt.Sprintf("%s=%s\n", TokenSecretField, fields[TokenSecretField])); err != nil {
		return nil, errors.WithStack(err)
	}
	//}

	if dk.NeedsStatsd() {
		if _, err := endpointBuf.WriteString(fmt.Sprintf("%s=%s\n", StatsdIngestUrl, fields[StatsdIngestUrl])); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	data := map[string][]byte{
		configFile:       endpointBuf.Bytes(),
		TokenSecretField: []byte(fields[TokenSecretField]),
	}
	return data, nil
}

func (g *EndpointSecretGenerator) PrepareFields(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string]string, error) {
	fields := make(map[string]string)

	//if dk.NeedsMetricsIngest() {
	var tokens corev1.Secret
	if err := g.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if token, ok := tokens.Data[dtclient.DynatraceDataIngestToken]; ok {
		fields[TokenSecretField] = string(token)
	}

	if diUrl, err := dataIngestUrl(dk); err != nil {
		return nil, err
	} else {
		fields[UrlSecretField] = diUrl
	}
	//}

	if dk.NeedsStatsd() {
		if statsdUrl, err := statsdIngestUrl(dk); err != nil {
			return nil, err
		} else {
			fields[StatsdIngestUrl] = statsdUrl
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
	apiUrl, err := url.Parse(dk.Spec.APIURL)
	if err != nil {
		return "", errors.WithMessage(err, "failed to parse DynaKube.spec.apiUrl")
	}

	tenant, err := extractTenant(apiUrl)
	if err != nil {
		return "", err
	}

	serviceName := capability.BuildServiceName(dk.Name, agcapability.MultiActiveGateName)
	return fmt.Sprintf("https://%s.%s/e/%s/api/v2/metrics/ingest", serviceName, dk.Namespace, tenant), nil
}

// TODO: Merge with DynaKube.TenantUUID()?
func extractTenant(url *url.URL) (string, error) {
	tenant := ""
	subdomains := strings.Split(url.Hostname(), ".")
	subpaths := strings.Split(url.Path, "/")
	//Path = '/e/<token>/api' -> ['', 'e',  '<tenant>', 'api']
	if len(subpaths) >= 4 && subpaths[1] == "e" && subpaths[3] == "api" {
		tenant = subpaths[2]
	} else if len(subdomains) >= 2 {
		tenant = subdomains[0]
	} else {
		return "", fmt.Errorf("failed to parse DynaKube.spec.apiUrl, unknown tenant")
	}
	return tenant, nil
}

func statsdIngestUrl(dk *dynatracev1beta1.DynaKube) (string, error) {
	serviceName := capability.BuildServiceName(dk.Name, agcapability.MultiActiveGateName)
	return fmt.Sprintf("%s.%s:%d", serviceName, dk.Namespace, agcapability.StatsdIngestPort), nil
}
