package extension

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/servicename"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconciler_prepareService(t *testing.T) {
	t.Run(`Check labels from service`, func(t *testing.T) {
		dk := createDynakube()

		fakeClient := fake.NewClient()
		r := &reconciler{client: fakeClient, apiReader: fakeClient, dk: dk, timeProvider: timeprovider.New()}
		svc, err := r.buildService()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			labels.AppComponentLabel: labels.ExtensionComponentLabel,
			labels.AppCreatedByLabel: dk.Name,
			labels.AppNameLabel:      version.AppName,
			labels.AppVersionLabel:   version.Version,
		}, svc.Labels)

		assert.Equal(t, map[string]string{
			labels.AppManagedByLabel: version.AppName,
			labels.AppCreatedByLabel: dk.Name,
			labels.AppNameLabel:      labels.ExtensionComponentLabel,
		}, svc.Spec.Selector)
	})
}

func TestFQDNNameGeneration(t *testing.T) {
	t.Run(`Check FQDN name generation`, func(t *testing.T) {
		dk := createDynakube()
		assert.Equal(t, "test-name-extensions-controller.test-namespace", servicename.BuildFQDN(dk))
	})
}
