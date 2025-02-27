package bootstrapperconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/enrichment/endpoint"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/curl"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretGenerator manages the bootstrapper init secret generation for the user namespaces.
type SecretGenerator struct {
	ctx       context.Context
	client    client.Client
	dtClient  dtclient.Client
	apiReader client.Reader
	namespace string
}

func NewBootstrapperInitGenerator(ctx context.Context, client client.Client, apiReader client.Reader, dtClient dtclient.Client, namespace string) *SecretGenerator {
	return &SecretGenerator{
		ctx:       ctx,
		client:    client,
		dtClient:  dtClient,
		apiReader: apiReader,
		namespace: namespace,
	}
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (s *SecretGenerator) GenerateForDynakube(dk *dynakube.DynaKube) error {
	log.Info("reconciling namespace bootstrapper init secret for", "dynakube", dk.Name)

	data, err := s.generate(s.ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	nsList, err := mapper.GetNamespacesForDynakube(s.ctx, s.apiReader, dk.Name)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(consts.BootstrapperInitSecretName, "", data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return err
	}

	err = k8ssecret.Query(s.client, s.apiReader, log).CreateOrUpdateForNamespaces(s.ctx, secret, nsList)
	if err != nil {
		return err
	}

	log.Info("done updating init secrets")

	return nil
}

func (s *SecretGenerator) Cleanup(namespaces []corev1.Namespace) error {
	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	return k8ssecret.Query(s.client, s.apiReader, log).DeleteForNamespaces(s.ctx, consts.BootstrapperInitSecretName, nsList)
}

// generate gets the necessary info the create the init secret data
func (s *SecretGenerator) generate(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {

	agCerts, err := dk.ActiveGateTLSCert(ctx, s.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	trustedCAs, err := dk.TrustedCAs(ctx, s.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	pmcSecret, err := s.preparePMC(ctx, *dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	endpointProperties, err := s.prepareEndpoints(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		pmc.InputFileName:        pmcSecret,
		ca.TrustedCertsInputFile: trustedCAs,
		ca.AgCertsInputFile:      agCerts,
		curl.InputFileName:       []byte{byte(dk.FeatureAgentInitialConnectRetry())},
		endpoint.InputFileName:   endpointProperties[endpoint.InputFileName],
	}, nil
}

func (s *SecretGenerator) prepareEndpoints(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	fields, err := s.prepareFieldsForEndpoints(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	endpointPropertiesBuilder := strings.Builder{}

	if dk.MetadataEnrichmentEnabled() {
		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", dtingestendpoint.MetricsUrlSecretField, fields[dtingestendpoint.MetricsUrlSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}

		if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", dtingestendpoint.MetricsTokenSecretField, fields[dtingestendpoint.MetricsTokenSecretField])); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	data := map[string][]byte{
		endpoint.InputFileName: bytes.NewBufferString(endpointPropertiesBuilder.String()).Bytes(),
	}

	return data, nil
}

func (s *SecretGenerator) prepareFieldsForEndpoints(ctx context.Context, dk *dynakube.DynaKube) (map[string]string, error) {
	fields := make(map[string]string)

	tokens, err := k8ssecret.Query(s.client, s.apiReader, log).Get(s.ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: s.namespace})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if dk.MetadataEnrichmentEnabled() {
		if token, ok := tokens.Data[dtclient.DataIngestToken]; ok {
			fields[dtingestendpoint.MetricsTokenSecretField] = string(token)
		} else {
			log.Info("data ingest token not found in secret", "dk", dk.Name)
		}

		if ingestUrl, err := dtingestendpoint.IngestUrlFor(dk); err != nil {
			return nil, err
		} else {
			fields[dtingestendpoint.MetricsUrlSecretField] = ingestUrl
		}
	}

	return fields, nil
}

func (s *SecretGenerator) preparePMC(ctx context.Context, dk dynakube.DynaKube) ([]byte, error) {
	pmc, err := s.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		conditions.SetDynatraceApiError(dk.Conditions(), processmoduleconfigsecret.PMCConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, s.apiReader, types.NamespacedName{
		Name:      dk.OneAgent().GetTenantSecret(),
		Namespace: dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeApiError(dk.Conditions(), processmoduleconfigsecret.PMCConditionType, err)

		return nil, err
	}

	pmc = pmc.
		AddHostGroup(dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if dk.NeedsOneAgentProxy() {
		proxy, err := dk.Proxy(ctx, s.apiReader)
		if err != nil {
			conditions.SetKubeApiError(dk.Conditions(), processmoduleconfigsecret.PMCConditionType, err)

			return nil, err
		}

		pmc.AddProxy(proxy)

		multiCap := capability.NewMultiCapability(&dk)
		dnsEntry := capability.BuildDNSEntryPointWithoutEnvVars(dk.Name, dk.Namespace, multiCap)

		if dk.FeatureNoProxy() != "" {
			dnsEntry += "," + dk.FeatureNoProxy()
		}

		pmc.AddNoProxy(dnsEntry)
	}

	marshaled, err := json.Marshal(pmc)
	if err != nil {
		log.Info("could not marshal process module config")

		return nil, err
	}

	return marshaled, err
}
