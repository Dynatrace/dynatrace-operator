package ingestendpoint

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	agcapability "github.com/Dynatrace/dynatrace-operator/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	UrlSecretField   = "DT_METRICS_INGEST_URL"
	TokenSecretField = "DT_METRICS_INGEST_API_TOKEN"
	configFile       = "endpoint.properties"
)

// EndpointSecretGenerator manages the mint endpoint secret generation for the user namespaces.
type EndpointSecretGenerator struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

func NewEndpointSecretGenerator(client client.Client, apiReader client.Reader, ns string, logger logr.Logger) *EndpointSecretGenerator {
	return &EndpointSecretGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
		logger:    logger,
	}
}

// GenerateForNamespace creates the data-ingest-endpoint secret for namespace while only having the name of the corresponding dynakube
// Used by the podInjection webhook in case the namespace lacks the secret.
func (g *EndpointSecretGenerator) GenerateForNamespace(ctx context.Context, dkName, targetNs string) (bool, error) {
	g.logger.Info("Reconciling data-ingest endpoint secret for", "namespace", targetNs)
	var dk dynatracev1beta1.DynaKube
	if err := g.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: g.namespace}, &dk); err != nil {
		return false, err
	}

	data, err := g.prepare(ctx, &dk)
	if err != nil {
		return false, err
	}
	return kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, SecretEndpointName, targetNs, data, corev1.SecretTypeOpaque, g.logger)
}

// GenerateForDynakube creates/updates the data-ingest-endpoint secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *EndpointSecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1beta1.DynaKube) (bool, error) {
	g.logger.Info("Reconciling data-ingest endpoint secret for", "dynakube", dk.Name)

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
		if upd, err := kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, SecretEndpointName, targetNs.Name, data, corev1.SecretTypeOpaque, g.logger); err != nil {
			return upd, err
		} else if upd {
			anyUpdate = true
		}
	}
	g.logger.Info("Done updating data-ingest endpoint secrets")
	return anyUpdate, nil
}

func (g *EndpointSecretGenerator) prepare(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	fields, err := g.PrepareFields(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var endpointBuf bytes.Buffer
	if _, err := endpointBuf.WriteString(fmt.Sprintf("%s=%s\n", UrlSecretField, fields[UrlSecretField])); err != nil {
		return nil, errors.WithStack(err)
	}
	if _, err := endpointBuf.WriteString(fmt.Sprintf("%s=%s\n", TokenSecretField, fields[TokenSecretField])); err != nil {
		return nil, errors.WithStack(err)
	}

	data := map[string][]byte{
		configFile: endpointBuf.Bytes(),
	}
	return data, nil
}

func (g *EndpointSecretGenerator) PrepareFields(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string]string, error) {
	var tokens corev1.Secret
	if err := g.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	dataIngestToken := ""
	if token, ok := tokens.Data[dtclient.DynatraceDataIngestToken]; ok {
		dataIngestToken = string(token)
	}

	diUrl, err := getDataIngestUrlFromDk(dk)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		UrlSecretField:   diUrl,
		TokenSecretField: dataIngestToken,
	}, nil
}

func getDataIngestUrlFromDk(dk *dynatracev1beta1.DynaKube) (string, error) {
	if dk.IsActiveGateMode(dynatracev1beta1.DataIngestCapability.ShortName) {
		return dataIngestUrlFromApiUrl(dk)
	} else if len(dk.Spec.APIURL) > 0 {
		return fmt.Sprintf("%s/v2/metrics/ingest", dk.Spec.APIURL), nil
	} else {
		return "", fmt.Errorf("failed to create data-ingest endpoint, DynaKube.spec.apiUrl is empty")
	}
}

func dataIngestUrlFromApiUrl(dk *dynatracev1beta1.DynaKube) (string, error) {
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
