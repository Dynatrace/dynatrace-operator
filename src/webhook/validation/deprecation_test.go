package validation

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

func TestDeprecationWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureEnableWebhookReinvocationPolicy: "test",
		}
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		})
	})

	t.Run(`warning present`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.DeprecatedFeatureFlagPrefix + dynatracev1beta1.AnnotationFeatureEnableWebhookReinvocationPolicy: "test",
		}
		assertAllowedResponseWithWarnings(t, 1, &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})
}
