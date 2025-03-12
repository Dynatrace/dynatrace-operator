package bootstrapperconfig

import (
	"context"
	"strconv"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/enrichment/endpoint"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/curl"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretGenerator manages the bootstrapper init secret generation for the user namespaces.
type SecretGenerator struct {
	client    client.Client
	dtClient  dtclient.Client
	apiReader client.Reader
}

func NewSecretGenerator(client client.Client, apiReader client.Reader, dtClient dtclient.Client) *SecretGenerator {
	return &SecretGenerator{
		client:    client,
		dtClient:  dtClient,
		apiReader: apiReader,
	}
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (s *SecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling namespace bootstrapper init secret for", "dynakube", dk.Name)

	data, err := s.generate(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	nsList, err := mapper.GetNamespacesForDynakube(ctx, s.apiReader, dk.Name)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(consts.BootstrapperInitSecretName, "", data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return err
	}

	err = k8ssecret.Query(s.client, s.apiReader, log).CreateOrUpdateForNamespaces(ctx, secret, nsList)
	if err != nil {
		return err
	}

	log.Info("done updating init secrets")

	return nil
}

func Cleanup(ctx context.Context, client client.Client, apiReader client.Reader, namespaces []corev1.Namespace) error {
	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	return k8ssecret.Query(client, apiReader, log).DeleteForNamespaces(ctx, consts.BootstrapperInitSecretName, nsList)
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

	initialConnectRetryMs := ""
	if dk.FeatureAgentInitialConnectRetry() > -1 {
		initialConnectRetryMs = strconv.Itoa(dk.FeatureAgentInitialConnectRetry())
	}

	return map[string][]byte{
		pmc.InputFileName:        pmcSecret,
		ca.TrustedCertsInputFile: trustedCAs,
		ca.AgCertsInputFile:      agCerts,
		curl.InputFileName:       []byte(initialConnectRetryMs),
		endpoint.InputFileName:   []byte(endpointProperties),
	}, nil
}
