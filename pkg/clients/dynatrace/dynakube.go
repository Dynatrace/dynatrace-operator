package dynatrace

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClientFromDynakube creates a new Dynatrace Client using the provided DynaKube configuration and tokens.
func NewClientFromDynakube(ctx context.Context, dk dynakube.DynaKube, apiToken string, paasToken string, apiReader client.Reader) (*ClientV2, error) {
	// If no separate PaaS token, use API token for PaaS operations
	if paasToken == "" {
		paasToken = apiToken
	}

	// Start building options
	options := []OptionV2{
		WithAPIToken(apiToken),
		WithPaasToken(paasToken),
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
	options = append(options, WithSkipCertificateValidation(dk.Spec.SkipCertCheck))

	// Configure trusted certificates
	if dk.Spec.TrustedCAs != "" {
		cm, err := getCertificateConfigmap(ctx, apiReader, dk.Spec.TrustedCAs, dk.Namespace)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// in PoC implementation there was
		// If we can't get system cert pool, create a new one
		// rootCAs = x509.NewCertPool() <- which differs from WithCerts implementation that always uses systemCertPool
		options = append(options, WithCerts([]byte(cm.Data[dynakube.TrustedCAKey])))
	}

	// Configure proxy settings
	if dk.HasProxy() {
		proxyURL, err := dk.Proxy(ctx, apiReader)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// how to get noProxy from dk?
		options = append(options, WithProxy(proxyURL, ""))
	}

	options = append(options, WithTimeout(30*time.Second))
	options = append(options, WithBaseURL(dk.Spec.APIURL))

	return NewClientV2(options...)
}

func getCertificateConfigmap(ctx context.Context, apiReader client.Reader, trustedCAs string, namespace string) (*corev1.ConfigMap, error) {
	certs := &corev1.ConfigMap{}
	if err := apiReader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: trustedCAs}, certs); err != nil {
		return nil, errors.WithMessage(err, "failed to get certificate configmap")
	}

	if certs.Data[dynakube.TrustedCAKey] == "" {
		return nil, errors.New("failed to extract certificate configmap field: missing field certs")
	}

	return certs, nil
}
