package extension

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconciler_prepareService(t *testing.T) {
	t.Run("Check labels from service", func(t *testing.T) {
		dk := createDynakube()

		fakeClient := fake.NewClient()
		r := &reconciler{client: fakeClient, apiReader: fakeClient, dk: dk, timeProvider: timeprovider.New()}
		svc, err := r.buildService()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			k8slabel.AppComponentLabel: k8slabel.ExtensionComponentLabel,
			k8slabel.AppCreatedByLabel: dk.Name,
			k8slabel.AppNameLabel:      version.AppName,
			k8slabel.AppVersionLabel:   version.Version,
		}, svc.Labels)

		assert.Equal(t, map[string]string{
			k8slabel.AppManagedByLabel: version.AppName,
			k8slabel.AppCreatedByLabel: dk.Name,
			k8slabel.AppNameLabel:      k8slabel.ExtensionComponentLabel,
		}, svc.Spec.Selector)
	})
}

func TestFQDNNameGeneration(t *testing.T) {
	t.Run("Check FQDN name generation", func(t *testing.T) {
		dk := createDynakube()
		assert.Equal(t, "test-name-extension-controller.test-namespace", dk.Extensions().GetServiceNameFQDN())
	})
}
