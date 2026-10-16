// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconciler_prepareService(t *testing.T) {
	t.Run("Check labels from service", func(t *testing.T) {
		dk := createDynakube()

		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient)
		svc, err := r.buildService(dk)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			k8slabel.AppName:         k8slabel.Extension,
			k8slabel.AppInstance:     dk.Name,
			k8slabel.AppManagedBy:    version.AppName,
			k8slabel.OperatorVersion: version.Version,
		}, svc.Labels)

		assert.Equal(t, map[string]string{
			k8slabel.AppName:      k8slabel.Extension,
			k8slabel.AppInstance:  dk.Name,
			k8slabel.AppManagedBy: version.AppName,
		}, svc.Spec.Selector)
	})
}

func TestFQDNNameGeneration(t *testing.T) {
	t.Run("Check FQDN name generation", func(t *testing.T) {
		dk := createDynakube()
		assert.Equal(t, "test-name-extension-controller.test-namespace", dk.Extensions().GetServiceNameFQDN())
	})
}
