// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DigestDelimiter = "@"
)

func addProxy(transport *http.Transport, proxy string, noProxy string) (*http.Transport, error) {
	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	proxyConfig := httpproxy.Config{
		HTTPProxy:  proxyURL.String(),
		HTTPSProxy: proxyURL.String(),
		NoProxy:    noProxy,
	}
	transport.Proxy = proxyWrapper(proxyConfig)

	return transport, nil
}

func proxyWrapper(proxyConfig httpproxy.Config) func(req *http.Request) (*url.URL, error) {
	proxyFunc := proxyConfig.ProxyFunc()

	return func(req *http.Request) (*url.URL, error) { return proxyFunc(req.URL) }
}

func addCertificates(transport *http.Transport, trustedCAs []byte) (*http.Transport, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read system certificates")
	}

	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		return nil, errors.New("failed to append custom certs")
	}

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} //nolint:gosec
	}

	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}

func addSkipCertCheck(transport *http.Transport, skipCertCheck bool) *http.Transport {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} //nolint:gosec
	}

	transport.TLSClientConfig.InsecureSkipVerify = skipCertCheck

	return transport
}

// PrepareTransportForDynaKube creates default http transport and add proxy or trustedCAs if any
func PrepareTransportForDynaKube(ctx context.Context, apiReader client.Reader, transport *http.Transport, dk *dynakube.DynaKube) (*http.Transport, error) {
	var (
		proxy      string
		trustedCAs []byte
		err        error
	)

	if dk.HasProxy() {
		proxy, err = dk.Proxy(ctx, apiReader)
		if err != nil {
			return nil, err
		}
	}

	if dk.Spec.TrustedCAs != "" {
		trustedCAs, err = dk.TrustedCAs(ctx, apiReader)
		if err != nil {
			return nil, err
		}
	}

	if proxy != "" {
		transport, err = addProxy(transport, proxy, dk.FF().GetNoProxy())
		if err != nil {
			return nil, errors.WithMessage(err, "failed to add proxy to default transport")
		}
	}

	if len(trustedCAs) > 0 {
		transport, err = addCertificates(transport, trustedCAs)
		if err != nil {
			return nil, err
		}
	}

	transport = addSkipCertCheck(transport, ptr.Deref(dk.Spec.SkipCertCheck, false))

	return transport, nil
}
