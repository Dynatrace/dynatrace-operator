package dynakube

import (
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingNamespaceSelector(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta2.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
						AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
							NamespaceSelector: metav1.LabelSelector{
								MatchLabels: dummyLabels,
							},
						},
					},
				},
			},
		},
			&dynatracev1beta2.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta2.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
							AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: dummyLabels,
								},
							},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
	t.Run(`invalid dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorConflictingNamespaceSelector},
			&dynatracev1beta2.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta2.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
							AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: dummyLabels,
								},
							},
						},
					},
				},
			},
			&dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflicting-dk",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta2.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
							AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
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
			assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-namespace-selector",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta2.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
							AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: map[string]string{
										"dummy": label,
									},
								},
							},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
		}
		// MatchExpressions
		assertAllowedResponse(t, &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid-namespace-selector",
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
						AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
							NamespaceSelector: metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "dummy",
										Operator: metav1.LabelSelectorOpIn,
										Values:   testsValidLabels,
									},
								},
							},
						},
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
			assertDeniedResponse(t,
				[]string{errorNamespaceSelectorMatchLabelsViolateLabelSpec},
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-namespace-selector",
						Namespace: testNamespace,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						APIURL: testApiUrl,
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
								AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
									NamespaceSelector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"dummy": label,
										},
									},
								},
							},
						},
					},
				}, &dummyNamespace, &dummyNamespace2)
		}
		// MatchExpressions
		assertDeniedResponse(t,
			[]string{errorNamespaceSelectorMatchLabelsViolateLabelSpec},
			&dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-namespace-selector",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta2.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
							AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "dummy",
											Operator: metav1.LabelSelectorOpIn,
											Values:   testsInvalidLabels,
										},
									},
								},
							},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
}
