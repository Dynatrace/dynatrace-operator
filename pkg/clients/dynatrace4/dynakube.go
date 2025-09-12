package dynatrace4

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dynatoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClientFromDynakube creates a new Dynatrace dtClient using the provided DynaKube configuration and tokens.
// This method replaces the builder pattern and directly creates a configured dtClient.
func NewClientFromDynakube(ctx context.Context, dk dynakube.DynaKube, tokens dynatoken.Tokens, apiReader client.Reader) (DtClient, error) {
	// Get token values
	apiToken := tokens.APIToken().Value
	paasToken := tokens.PaasToken().Value
	dataIngestToken := tokens.DataIngestToken().Value

	// If no separate PaaS token, use API token for PaaS operations
	if paasToken == "" {
		paasToken = apiToken
	}

	// Start building options
	options := []Option{
		WithAPIToken(apiToken),
		WithPaasToken(paasToken),
		WithDataIngestToken(dataIngestToken),
	}

	// Configure network zone
	if dk.Spec.NetworkZone != "" {
		options = append(options, WithNetworkZone(dk.Spec.NetworkZone))
	}

	// Configure host group
	if dk.OneAgent().GetHostGroup() != "" {
		options = append(options, WithHostGroup(dk.OneAgent().GetHostGroup()))
	}

	// Configure TLS settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: dk.Spec.SkipCertCheck, //nolint:gosec
	}

	// Configure trusted certificates
	if dk.Spec.TrustedCAs != "" {
		err := appendTrustedCerts(ctx, apiReader, tlsConfig, dk.Spec.TrustedCAs, dk.Namespace)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Configure proxy settings and HTTP dtClient
	var httpClient *http.Client

	if dk.HasProxy() {
		proxyURL, err := dk.Proxy(ctx, apiReader)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
		}

		if proxyURL != "" {
			parsedProxyURL, err := url.Parse(proxyURL)
			if err != nil {
				return nil, fmt.Errorf("invalid proxy URL: %w", err)
			}

			transport.Proxy = http.ProxyURL(parsedProxyURL)
		}

		httpClient = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	} else {
		// No proxy, but still apply TLS config
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
			Timeout: 30 * time.Second,
		}
	}

	options = append(options, WithHTTPClient(httpClient))

	// Create the dtClient
	return newClient(dk.Spec.APIURL, options...)
}

// appendTrustedCerts adds trusted certificates to the TLS configuration
func appendTrustedCerts(ctx context.Context, apiReader client.Reader, tlsConfig *tls.Config, trustedCAs string, namespace string) error {
	if trustedCAs == "" {
		return nil
	}

	certs := &corev1.ConfigMap{}
	if err := apiReader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: trustedCAs}, certs); err != nil {
		return errors.WithMessage(err, "failed to get certificate configmap")
	}

	if certs.Data[dynakube.TrustedCAKey] == "" {
		return errors.New("failed to extract certificate configmap field: missing field certs")
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		// If we can't get system cert pool, create a new one
		rootCAs = x509.NewCertPool()
	}

	if ok := rootCAs.AppendCertsFromPEM([]byte(certs.Data[dynakube.TrustedCAKey])); !ok {
		return errors.New("failed to append custom certs")
	}

	tlsConfig.RootCAs = rootCAs

	return nil
}
