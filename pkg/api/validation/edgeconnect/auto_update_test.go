// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

func Test_deprecatedAutoUpdate(t *testing.T) {
	newEdgeConnect := func(autoUpdate *bool) *edgeconnect.EdgeConnect {
		return &edgeconnect.EdgeConnect{
			Name:      testName,
			Namespace: testNamespace,
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "tenantid" + allowedSuffix[0],
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     testValidOAuthEndpoint,
					Resource:     "resource",
				},
				AutoUpdate: autoUpdate, //nolint:staticcheck // SA1019: testing deprecated field
			},
		}
	}

	t.Run("no warning when autoUpdate is not set", func(t *testing.T) {
		assertAllowed(t, newEdgeConnect(nil))
	})
	t.Run("warning when autoUpdate is true", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1, newEdgeConnect(new(true)))
	})
	t.Run("warning when autoUpdate is false", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1, newEdgeConnect(new(false)))
	})
}
