package validation

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
)

func TestPreviewWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1.ApplicationMonitoringSpec{},
				},
			},
		})
	})
}
