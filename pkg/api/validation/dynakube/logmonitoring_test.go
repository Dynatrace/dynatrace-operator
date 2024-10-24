package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIgnoredLogMonitoringTemplates(t *testing.T) {
	t.Run("no warning if logMonitoring template section is empty", func(t *testing.T) {
		dk := createStandaloneLogMonitoringDynakube(testName, "")
		dk.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{}
		assertAllowedWithoutWarnings(t, dk, &defaultCSIDaemonSet)
	})
	t.Run("warning if logMonitoring template section is not empty", func(t *testing.T) {
		dk := createStandaloneLogMonitoringDynakube(testName, "something")
		dk.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{}
		assertAllowedWithWarnings(t, 1, dk, &defaultCSIDaemonSet)
	})
}

func createStandaloneLogMonitoringDynakube(name, nodeSelector string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:        testApiUrl,
			LogMonitoring: &logmonitoring.Spec{},
		},
	}

	if nodeSelector != "" {
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			NodeSelector: map[string]string{"node": nodeSelector},
		}
	}

	return dk
}
