package validation

import (
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestDeprecationWarning(t *testing.T) {
	t.Run(`no warning`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureDisableWebhookReinvocationPolicy: "true",
		}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithoutWarnings(t, dynakube)
		assert.True(t, dynakube.FeatureDisableWebhookReinvocationPolicy())
	})

	t.Run(`warning present`, func(t *testing.T) {
		dynakubeMeta := defaultDynakubeObjectMeta
		split := strings.Split(dynatracev1beta1.AnnotationFeatureDisableWebhookReinvocationPolicy, "/")
		postFix := split[1]
		dynakubeMeta.Annotations = map[string]string{
			dynatracev1beta1.DeprecatedFeatureFlagPrefix + postFix: "true",
		}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: dynakubeMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		assertAllowedResponseWithWarnings(t, 1, dynakube)
		assert.True(t, dynakube.FeatureDisableWebhookReinvocationPolicy())
	})
}
