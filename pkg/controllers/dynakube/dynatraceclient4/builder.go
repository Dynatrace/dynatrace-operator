package dynatraceclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient4 "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder interface {
	SetDynakube(dk dynakube.DynaKube) Builder
	SetTokens(tokens token.Tokens) Builder
	Build(ctx context.Context) (*dtclient4.Client, error)
}

type builder struct {
	apiReader client.Reader
	tokens    token.Tokens
	dk        dynakube.DynaKube
}

func NewBuilder(apiReader client.Reader) Builder {
	return builder{
		apiReader: apiReader,
	}
}

func (dynatraceClientBuilder builder) SetDynakube(dk dynakube.DynaKube) Builder {
	dynatraceClientBuilder.dk = dk

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) SetTokens(tokens token.Tokens) Builder {
	dynatraceClientBuilder.tokens = tokens

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) getTokens() token.Tokens {
	if dynatraceClientBuilder.tokens == nil {
		dynatraceClientBuilder.tokens = token.Tokens{}
	}

	return dynatraceClientBuilder.tokens
}

// appendTrustedCerts adds trusted certificates to the TLS configuration
func (dynatraceClientBuilder builder) appendTrustedCerts(ctx context.Context, tlsConfig *tls.Config, trustedCAs string, namespace string) error {
	if trustedCAs == "" {
		return nil
	}

	certs := &corev1.ConfigMap{}
	if err := dynatraceClientBuilder.apiReader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: trustedCAs}, certs); err != nil {
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

// Build creates a new Dynatrace client using the settings configured on the given instance.
func (dynatraceClientBuilder builder) Build(ctx context.Context) (*dtclient4.Client, error) {
	// Read tokens if not already set
	var tokens token.Tokens
	if dynatraceClientBuilder.tokens == nil {
		tokenReader := token.NewReader(dynatraceClientBuilder.apiReader, &dynatraceClientBuilder.dk)
		var err error
		tokens, err = tokenReader.ReadTokens(ctx)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		tokens = dynatraceClientBuilder.tokens
	}

	// Get token values
	apiToken := tokens.APIToken().Value
	paasToken := tokens.PaasToken().Value
	dataIngestToken := tokens.DataIngestToken().Value

	// If no separate PaaS token, use API token for PaaS operations
	if paasToken == "" {
		paasToken = apiToken
	}

	// Start building options
	options := []dtclient4.Option{
		dtclient4.WithAPIToken(apiToken),
		dtclient4.WithPaasToken(paasToken),
		dtclient4.WithDataIngestToken(dataIngestToken),
	}

	// Configure network zone
	if dynatraceClientBuilder.dk.Spec.NetworkZone != "" {
		options = append(options, dtclient4.WithNetworkZone(dynatraceClientBuilder.dk.Spec.NetworkZone))
	}

	// Configure host group
	if dynatraceClientBuilder.dk.OneAgent().GetHostGroup() != "" {
		options = append(options, dtclient4.WithHostGroup(dynatraceClientBuilder.dk.OneAgent().GetHostGroup()))
	}

	// Configure TLS settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: dynatraceClientBuilder.dk.Spec.SkipCertCheck, //nolint:gosec
	}

	// Configure trusted certificates
	if dynatraceClientBuilder.dk.Spec.TrustedCAs != "" {
		err := dynatraceClientBuilder.appendTrustedCerts(ctx, tlsConfig, dynatraceClientBuilder.dk.Spec.TrustedCAs, dynatraceClientBuilder.dk.Namespace)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Configure proxy settings and HTTP client
	var httpClient *http.Client
	if dynatraceClientBuilder.dk.HasProxy() {
		proxyURL, err := dynatraceClientBuilder.dk.Proxy(ctx, dynatraceClientBuilder.apiReader)
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

	options = append(options, dtclient4.WithHTTPClient(httpClient))

	// Create the client
	return dtclient4.NewClient(dynatraceClientBuilder.dk.Spec.APIURL, options...)
}
