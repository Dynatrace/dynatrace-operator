package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube" //nolint:staticcheck
)

func TestPreviewWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
				},
			},
		})
	})
}
