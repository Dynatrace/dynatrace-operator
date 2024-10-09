/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dynakube

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ProxyKey   = "proxy"
	NoProxyKey = "noProxy"
)

func (dk *DynaKube) Proxy(ctx context.Context, kubeReader client.Reader) (string, error) {
	if dk.Spec.Proxy == nil {
		return "", nil
	}

	if dk.Spec.Proxy.Value != "" {
		return dk.Spec.Proxy.Value, nil
	} else if dk.Spec.Proxy.ValueFrom != "" {
		return dk.proxyUrlFromUserSecret(ctx, kubeReader)
	}

	return "", nil
}

func (dk *DynaKube) HasProxy() bool {
	return dk.Spec.Proxy != nil && (dk.Spec.Proxy.Value != "" || dk.Spec.Proxy.ValueFrom != "")
}

func (dk *DynaKube) NeedsCustomNoProxy() bool {
	return dk.HasProxy() && dk.FeatureNoProxy() != ""
}

func (dk *DynaKube) NeedsActiveGateProxy() bool {
	return !dk.FeatureActiveGateIgnoreProxy() && dk.HasProxy() && dk.ActiveGate().IsEnabled()
}

func (dk *DynaKube) NeedsOneAgentProxy() bool {
	return !dk.FeatureOneAgentIgnoreProxy() && dk.HasProxy()
}

func (dk *DynaKube) proxyUrlFromUserSecret(ctx context.Context, kubeReader client.Reader) (string, error) {
	secretName := dk.Spec.Proxy.ValueFrom

	var proxySecret corev1.Secret

	err := kubeReader.Get(ctx, client.ObjectKey{Name: secretName, Namespace: dk.Namespace}, &proxySecret)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("failed to get proxy from %s secret", secretName))
	}

	proxy, hasKey := proxySecret.Data[ProxyKey]
	if !hasKey {
		err := errors.Errorf("missing token %s in proxy secret %s", ProxyKey, secretName)

		return "", err
	}

	return string(proxy), nil
}
