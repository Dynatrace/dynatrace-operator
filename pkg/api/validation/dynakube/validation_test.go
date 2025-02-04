package validation

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testApiUrl    = "https://f.q.d.n/api"
)

var defaultDynakubeObjectMeta = metav1.ObjectMeta{
	Name:      testName,
	Namespace: testNamespace,
}

var dummyLabels = map[string]string{
	"dummy": "label",
}

var dummyNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "dummy",
		Labels: dummyLabels,
	},
}

var dummyLabels2 = map[string]string{
	"dummy": "label",
}

var dummyNamespace2 = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "dummy2",
		Labels: dummyLabels2,
	},
}

func TestDynakubeValidator_Handle(t *testing.T) {
	t.Run("valid dynakube specs", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
							},
						},
						AppInjectionSpec: oneagent.AppInjectionSpec{
							NamespaceSelector: metav1.LabelSelector{
								MatchLabels: dummyLabels,
							},
						},
					},
				},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
						activegate.KubeMonCapability.DisplayName,
						activegate.MetricsIngestCapability.DisplayName,
					},
				},
			},
		},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							HostInjectSpec: oneagent.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "2",
								},
							},
							AppInjectionSpec: oneagent.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: dummyLabels2,
								},
							},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
	t.Run("conflicting dynakube specs", func(t *testing.T) {
		setupDisabledCSIEnv(t)
		assertDenied(t,
			[]string{
				errorCSIModuleRequired,
				errorNoApiUrl,
				errorConflictingNamespaceSelector,
				fmt.Sprintf(errorDuplicateActiveGateCapability, activegate.KubeMonCapability.DisplayName),
				fmt.Sprintf(errorInvalidActiveGateCapability, "me dumb"),
				fmt.Sprintf(errorNodeSelectorConflict, "conflict2")},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: "",
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							AppInjectionSpec: oneagent.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: dummyLabels,
								},
							},
						},
					},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
							activegate.KubeMonCapability.DisplayName,
							"me dumb",
						},
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict1",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
							AppInjectionSpec: oneagent.AppInjectionSpec{
								NamespaceSelector: metav1.LabelSelector{
									MatchLabels: dummyLabels,
								},
							},
						},
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict2",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
}

func assertDenied(t *testing.T, errMessages []string, dk *dynakube.DynaKube, other ...client.Object) {
	_, err := runValidators(dk, other...)
	require.Error(t, err)

	for _, errMsg := range errMessages {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func assertUpdateDenied(t *testing.T, errMessages []string, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube, other ...client.Object) {
	_, err := runUpdateValidators(oldDk, newDk, other...)
	require.Error(t, err)

	for _, errMsg := range errMessages {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func assertAllowedWithoutWarnings(t *testing.T, dk *dynakube.DynaKube, other ...client.Object) {
	warnings, _ := assertAllowed(t, dk, other...)
	assert.Empty(t, warnings)
}

func assertAllowedWithWarnings(t *testing.T, warningAmount int, dk *dynakube.DynaKube, other ...client.Object) {
	warnings, _ := assertAllowed(t, dk, other...)
	assert.Len(t, warnings, warningAmount)
}

func assertAllowed(t *testing.T, dk *dynakube.DynaKube, other ...client.Object) (admission.Warnings, error) {
	warnings, err := runValidators(dk, other...)
	assert.NoError(t, err)

	return warnings, err
}

func assertUpdateAllowed(t *testing.T, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube, other ...client.Object) (admission.Warnings, error) {
	warnings, err := runUpdateValidators(oldDk, newDk, other...)
	assert.NoError(t, err)

	return warnings, err
}

func assertUpdateAllowedWithoutWarnings(t *testing.T, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube, other ...client.Object) {
	warnings, _ := assertUpdateAllowed(t, oldDk, newDk, other...)
	assert.Empty(t, warnings)
}

func runValidators(dk *dynakube.DynaKube, other ...client.Object) (admission.Warnings, error) {
	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &Validator{
		apiReader: clt,
		cfg:       &rest.Config{},
		modules:   installconfig.GetModules(),
	}

	return validator.ValidateCreate(context.Background(), dk)
}

func runUpdateValidators(oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube, other ...client.Object) (admission.Warnings, error) {
	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &Validator{
		apiReader: clt,
		cfg:       &rest.Config{},
		modules:   installconfig.GetModules(),
	}

	return validator.ValidateUpdate(context.Background(), oldDk, newDk)
}
