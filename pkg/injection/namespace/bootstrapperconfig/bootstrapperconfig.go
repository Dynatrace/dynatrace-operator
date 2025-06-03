package bootstrapperconfig

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/enrichment/endpoint"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/curl"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper/download"
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

// SecretGenerator manages the bootstrapper init secret generation for the user namespaces.
type SecretGenerator struct {
	client       client.Client
	dtClient     dtclient.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider
}

func NewSecretGenerator(client client.Client, apiReader client.Reader, dtClient dtclient.Client) *SecretGenerator {
	return &SecretGenerator{
		client:       client,
		dtClient:     dtClient,
		apiReader:    apiReader,
		timeProvider: timeprovider.New(),
	}
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (s *SecretGenerator) GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling namespace bootstrapper init secret for", "dynakube", dk.Name)

	data, err := s.generateConfig(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.createSourceForWebhook(ctx, dk, GetSourceConfigSecretName(dk.Name), data)
	if err != nil {
		return err
	}

	certs, err := s.generateCerts(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.createSourceForWebhook(ctx, dk, GetSourceCertsSecretName(dk.Name), certs)
	if err != nil {
		return err
	}

	nsList, err := mapper.GetNamespacesForDynakube(ctx, s.apiReader, dk.Name)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return errors.WithStack(err)
	}

	err = s.createSecretForNSlist(ctx, consts.BootstrapperInitSecretName, nsList, dk, data)
	if err != nil {
		return errors.WithStack(err)
	}

	return s.createSecretForNSlist(ctx, consts.BootstrapperInitCertsSecretName, nsList, dk, certs)
}

func (s *SecretGenerator) createSecretForNSlist(ctx context.Context, secretName string, nsList []corev1.Namespace, dk *dynakube.DynaKube, data map[string][]byte) error {
	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(secretName, "", data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), ConditionType, err)

		return err
	}

	err = k8ssecret.Query(s.client, s.apiReader, log).CreateOrUpdateForNamespaces(ctx, secret, nsList)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return err
	}

	log.Info("done updating init secrets")
	conditions.SetSecretCreatedOrUpdated(dk.Conditions(), ConditionType, GetSourceConfigSecretName(dk.Name))

	return nil
}

func Cleanup(ctx context.Context, client client.Client, apiReader client.Reader, namespaces []corev1.Namespace, dk *dynakube.DynaKube) error {
	defer meta.RemoveStatusCondition(dk.Conditions(), ConditionType)

	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	err := k8ssecret.Query(client, apiReader, log).DeleteForNamespace(ctx, GetSourceConfigSecretName(dk.Name), dk.Namespace)
	if err != nil {
		log.Error(err, "failed to delete the source bootstrapper-config secret", "name", GetSourceConfigSecretName(dk.Name))
	}

	return k8ssecret.Query(client, apiReader, log).DeleteForNamespaces(ctx, consts.BootstrapperInitSecretName, nsList)
}

// generate gets the necessary info the create the init secret data
func (s *SecretGenerator) generateConfig(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	data := map[string][]byte{}

	endpointProperties, err := s.prepareEndpoints(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(endpointProperties) != 0 {
		data[endpoint.InputFileName] = []byte(endpointProperties)
	}

	var tokens corev1.Secret
	if err := s.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	var proxy string
	if dk.NeedsOneAgentProxy() {
		proxy, err = dk.Proxy(ctx, s.apiReader)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	downloadConfigJSON := download.Config{
		URL:           dk.Spec.APIURL,
		APIToken:      string(tokens.Data[dtclient.ApiToken]),
		Proxy:         proxy,
		NoProxy:       dk.FF().GetNoProxy(),
		NetworkZone:   dk.Spec.NetworkZone,
		HostGroup:     dk.OneAgent().GetHostGroup(),
		SkipCertCheck: dk.Spec.SkipCertCheck,
	}

	downloadConfigBytes, err := json.Marshal(downloadConfigJSON)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return nil, errors.WithStack(err)
	}

	data[download.InputFileName] = downloadConfigBytes

	pmcSecret, err := s.preparePMC(ctx, dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(pmcSecret) != 0 {
		data[pmc.InputFileName] = pmcSecret
	}

	if dk.FF().GetAgentInitialConnectRetry(dk.Spec.EnableIstio) > -1 {
		initialConnectRetryMs := strconv.Itoa(dk.FF().GetAgentInitialConnectRetry(dk.Spec.EnableIstio))
		data[curl.InputFileName] = []byte(initialConnectRetryMs)
	}

	return data, err
}

// generateCerts gets the necessary info they create the init certs secret data
func (s *SecretGenerator) generateCerts(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	data := map[string][]byte{}

	agCerts, err := dk.ActiveGateTLSCert(ctx, s.apiReader)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return nil, errors.WithStack(err)
	}

	if len(agCerts) != 0 {
		data[ca.AgCertsInputFile] = agCerts
	}

	trustedCAs, err := dk.TrustedCAs(ctx, s.apiReader)

	if len(trustedCAs) != 0 {
		data[ca.TrustedCertsInputFile] = trustedCAs
	}

	return data, err
}
