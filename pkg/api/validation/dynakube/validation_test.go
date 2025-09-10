package validation

import (
	"context"
	"fmt"
	v1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"

	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
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
	testAPIURL    = "https://f.q.d.n/api"
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
				APIURL: testAPIURL,
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
					APIURL: testAPIURL,
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
				errorNoAPIURL,
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
					APIURL: testAPIURL,
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
					APIURL: testAPIURL,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
}

func Test_getDynakube(t *testing.T) {
	t.Run("v1beta5 to latest", func(t *testing.T) {
		v1beta5Dk := &v1beta5.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: v1beta5.DynaKubeSpec{
				APIURL: testAPIURL,
			},
		}

		dk, err := getDynakube(v1beta5Dk)
		require.NoError(t, err)

		assert.Equal(t, v1beta5Dk.Name, dk.Name)
		assert.Equal(t, v1beta5Dk.Namespace, dk.Namespace)
		assert.Equal(t, v1beta5Dk.Spec.APIURL, dk.Spec.APIURL)
	})

	t.Run("v1beta4 to latest", func(t *testing.T) {
		v1beta4Dk := &v1beta4.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: v1beta4.DynaKubeSpec{
				APIURL: testAPIURL,
			},
		}

		dk, err := getDynakube(v1beta4Dk)
		require.NoError(t, err)

		assert.Equal(t, v1beta4Dk.Name, dk.Name)
		assert.Equal(t, v1beta4Dk.Namespace, dk.Namespace)
		assert.Equal(t, v1beta4Dk.Spec.APIURL, dk.Spec.APIURL)
	})

	t.Run("v1beta3 to latest", func(t *testing.T) {
		v1beta3Dk := &v1beta3.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: v1beta3.DynaKubeSpec{
				APIURL: testAPIURL,
			},
		}

		dk, err := getDynakube(v1beta3Dk)
		require.NoError(t, err)

		assert.Equal(t, v1beta3Dk.Name, dk.Name)
		assert.Equal(t, v1beta3Dk.Namespace, dk.Namespace)
		assert.Equal(t, v1beta3Dk.Spec.APIURL, dk.Spec.APIURL)
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
