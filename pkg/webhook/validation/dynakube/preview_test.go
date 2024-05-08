package dynakube

import (
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
)

func TestPreviewWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta2.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{},
				},
			},
		})
	})
}
