package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube" //nolint:staticcheck
	v1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingNamespaceSelector(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				NamespaceSelector: metav1.LabelSelector{
					MatchLabels: dummyLabels,
				},
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						AppInjectionSpec: dynakube.AppInjectionSpec{},
					},
				},
			},
		},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels,
					},
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
							AppInjectionSpec: dynakube.AppInjectionSpec{},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
	t.Run(`invalid dynakube specs`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorConflictingNamespaceSelector},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels,
					},
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
							AppInjectionSpec: dynakube.AppInjectionSpec{},
						},
					},
				},
			},
			&v1beta3.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflicting-dk",
					Namespace: testNamespace,
				},
				Spec: v1beta3.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: v1beta3.OneAgentSpec{
						ApplicationMonitoring: &v1beta3.ApplicationMonitoringSpec{
							AppInjectionSpec: v1beta3.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: dummyLabels,
								},
							},
						},
					},
				},
			}, &defaultCSIDaemonSet, &dummyNamespace)
	})
	t.Run("validate namespaceSelector to be a valid label according to spec", func(t *testing.T) {
		testsValidLabels := []string{
			"",
			"a",
			"short",
			"WithUpperCase",
			"contains123",
			"label-with-Dash",
			"label_with_underscore",
			"label.with.dotttses",
			"label.with.dotttses-567567",
		}
		// MatchLabels
		for _, label := range testsValidLabels {
			assertAllowed(t, &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-namespace-selector",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"dummy": label,
						},
					},
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
							AppInjectionSpec: dynakube.AppInjectionSpec{},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
		}
		// MatchExpressions
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-namespace-selector",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				NamespaceSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "dummy",
							Operator: metav1.LabelSelectorOpIn,
							Values:   testsValidLabels,
						},
					},
				},
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						AppInjectionSpec: dynakube.AppInjectionSpec{},
					},
				},
			},
		}, &dummyNamespace, &dummyNamespace2)

		testsInvalidLabels := []string{
			"name%",
			"name/",
			"AMuchTooLongLabelThatGoesOverSixtyThreeCharactersAndSoIsInvalidAccordingToSpec",
		}
		for _, label := range testsInvalidLabels {
			// MatchLabels
			assertDenied(t,
				[]string{errorNamespaceSelectorMatchLabelsViolateLabelSpec},
				&dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-namespace-selector",
						Namespace: testNamespace,
					},
					Spec: dynakube.DynaKubeSpec{
						APIURL: testApiUrl,
						NamespaceSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"dummy": label,
							},
						},
						OneAgent: dynakube.OneAgentSpec{
							ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
								AppInjectionSpec: dynakube.AppInjectionSpec{},
							},
						},
					},
				}, &dummyNamespace, &dummyNamespace2)
		}
		// MatchExpressions
		assertDenied(t,
			[]string{errorNamespaceSelectorMatchLabelsViolateLabelSpec},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-namespace-selector",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "dummy",
								Operator: metav1.LabelSelectorOpIn,
								Values:   testsInvalidLabels,
							},
						},
					},
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
							AppInjectionSpec: dynakube.AppInjectionSpec{},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
}
