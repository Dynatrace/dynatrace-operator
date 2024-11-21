package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIgnoredLogMonitoringTemplate(t *testing.T) {
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

func TestMissingLogMonitoringImage(t *testing.T) {
	t.Run("both standalone log monitoring and image ref set", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: &logmonitoring.TemplateSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
								Tag:        "version",
							},
						},
					},
				},
			})
	})

	t.Run("standalone log monitoring but missing image", func(t *testing.T) {
		assertDenied(t,
			[]string{errorLogMonitoringMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
				},
			})
	})

	t.Run("standalone log monitoring and only image repository set", func(t *testing.T) {
		assertDenied(t,
			[]string{errorLogMonitoringMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: &logmonitoring.TemplateSpec{
							ImageRef: image.Ref{
								Repository: "repo/image",
							},
						},
					},
				},
			})
	})

	t.Run("kspm enabled and only image repository tag", func(t *testing.T) {
		assertDenied(t,
			[]string{errorLogMonitoringMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: &logmonitoring.TemplateSpec{
							ImageRef: image.Ref{
								Tag: "version",
							},
						},
					},
				},
			})
	})
}
