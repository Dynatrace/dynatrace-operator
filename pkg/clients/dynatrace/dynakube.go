// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynatrace

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientFactory func(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, apiToken, paasToken, userAgentSuffix string, options ...Option) (*Client, error)

// NewClientFromDynakube creates a new Dynatrace dtClient using the provided DynaKube configuration and tokens.
func NewClientFromDynakube(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, apiToken, paasToken, userAgentSuffix string, options ...Option) (*Client, error) {
	opts, err := combinedOptions(ctx, apiReader, dk, apiToken, paasToken, userAgentSuffix, options...)
	if err != nil {
		return nil, err
	}

	return NewClient(opts...)
}

func combinedOptions(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, apiToken, paasToken, userAgentSuffix string, options ...Option) ([]Option, error) {
	opts, err := optionsFromDynakube(ctx, apiReader, dk, apiToken, paasToken, userAgentSuffix)
	if err != nil {
		return nil, err
	}

	opts = append(opts, options...)

	return opts, nil
}

func optionsFromDynakube(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, apiToken, paasToken, userAgentSuffix string) ([]Option, error) {
	options := []Option{
		WithBaseURL(dk.APIURL()),
		WithAPIToken(apiToken),
		WithPaasToken(paasToken),
		WithSkipCertificateValidation(ptr.Deref(dk.Spec.SkipCertCheck, false)),
		WithUserAgentSuffix(userAgentSuffix),
		WithCacheTTL(dk.APIRequestThreshold()),
	}

	if dk.Spec.NetworkZone != "" {
		options = append(options, WithNetworkZone(dk.Spec.NetworkZone))
	}

	if dk.OneAgent().GetHostGroup() != "" {
		options = append(options, WithHostGroup(dk.OneAgent().GetHostGroup()))
	}

	if dk.Spec.TrustedCAs != "" {
		certs := &corev1.ConfigMap{}
		if err := apiReader.Get(ctx, client.ObjectKey{Namespace: dk.Namespace, Name: dk.Spec.TrustedCAs}, certs); err != nil {
			return nil, errors.WithMessage(err, "failed to get certificate configmap")
		}

		if certs.Data[dynakube.TrustedCAKey] == "" {
			return nil, errors.New("failed to extract certificate configmap field: missing field certs")
		}

		options = append(options, WithCerts([]byte(certs.Data[dynakube.TrustedCAKey])))
	}

	if dk.HasProxy() {
		proxyURL, err := dk.Proxy(ctx, apiReader)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		options = append(options, WithProxy(proxyURL, dk.FF().GetNoProxy()))
	}

	return options, nil
}
