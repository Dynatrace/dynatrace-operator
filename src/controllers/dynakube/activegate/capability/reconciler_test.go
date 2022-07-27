package capability

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/customproperties"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testValue     = "test-value"
	testUID       = "test-uid"
	testNamespace = "test-namespace"
)

var metricsCapability = capability.NewRoutingCapability(
	&dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			Routing: dynatracev1beta1.RoutingSpec{
				Enabled: true,
			},
		},
	},
)

type DtCapability = dynatracev1beta1.CapabilityDisplayName

func testRemoveCapability(capabilities []DtCapability, removeMe DtCapability) []DtCapability {
	for i, capability := range capabilities {
		if capability == removeMe {
			return append(capabilities[:i], capabilities[i+1:]...)
		}
	}
	return capabilities
}

func testSetCapability(instance *dynatracev1beta1.DynaKube, capability dynatracev1beta1.ActiveGateCapability, wantEnabled bool) {
	hasEnabled := instance.IsActiveGateMode(capability.DisplayName)
	capabilities := &instance.Spec.ActiveGate.Capabilities

	if wantEnabled && !hasEnabled {
		*capabilities = append(*capabilities, capability.DisplayName)
	}

	if !wantEnabled && hasEnabled {
		*capabilities = testRemoveCapability(*capabilities, capability.DisplayName)
	}
}

func TestNewReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) *Reconciler {
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			},
		}).
		Build()
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
		},
	}
	r := NewReconciler(metricsCapability, clt, clt, scheme.Scheme, instance)
	require.NotNil(t, r)
	require.NotNil(t, r.Client)
	require.NotNil(t, r.Instance)
	require.NotEmpty(t, r.Instance.ObjectMeta.Name)

	return r
}

func TestReconcile(t *testing.T) {
	assertStatefulSetExists := func(r *Reconciler) *appsv1.StatefulSet {
		statefulSet := new(appsv1.StatefulSet)
		assert.NoError(t, r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Instance.Namespace}, statefulSet))
		assert.NotNil(t, statefulSet)
		return statefulSet
	}
	assertServiceExists := func(r *Reconciler) *corev1.Service {
		svc := new(corev1.Service)
		assert.NoError(t, r.Get(context.TODO(), client.ObjectKey{Name: BuildServiceName(r.Instance.Name, r.ShortName()), Namespace: r.Instance.Namespace}, svc))
		assert.NotNil(t, svc)
		return svc
	}
	reconcileAndExpectUpdate := func(r *Reconciler, updateExpected bool) {
		update, err := r.Reconcile()
		assert.NoError(t, err)
		assert.Equal(t, updateExpected, update)
	}
	setStatsdCapability := func(r *Reconciler, wantEnabled bool) {
		testSetCapability(r.Instance, dynatracev1beta1.StatsdIngestCapability, wantEnabled)
	}
	setMetricsIngestCapability := func(r *Reconciler, wantEnabled bool) {
		testSetCapability(r.Instance, dynatracev1beta1.MetricsIngestCapability, wantEnabled)
	}

	agIngestServicePort := corev1.ServicePort{
		Name:       capability.HttpsServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       capability.HttpsServicePort,
		TargetPort: intstr.FromString(capability.HttpsServicePortName),
	}
	agIngestHttpServicePort := corev1.ServicePort{
		Name:       capability.HttpServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       capability.HttpServicePort,
		TargetPort: intstr.FromString(capability.HttpServicePortName),
	}
	statsdIngestServicePort := corev1.ServicePort{
		Name:       capability.StatsdIngestPortName,
		Protocol:   corev1.ProtocolUDP,
		Port:       capability.StatsdIngestPort,
		TargetPort: intstr.FromString(capability.StatsdIngestTargetPort),
	}

	t.Run(`reconcile custom properties`, func(t *testing.T) {
		r := createDefaultReconciler(t)

		metricsCapability.Properties().CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}
		// Reconcile twice since service is created before the stateful set is
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)

		var customProperties corev1.Secret
		err := r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + metricsCapability.ShortName() + "-" + customproperties.Suffix, Namespace: r.Instance.Namespace}, &customProperties)
		assert.NoError(t, err)
		assert.NotNil(t, customProperties)
		assert.Contains(t, customProperties.Data, customproperties.DataKey)
		assert.Equal(t, testValue, string(customProperties.Data[customproperties.DataKey]))
	})
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		// Reconcile twice since service is created before the stateful set is
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)

		statefulSet := assertStatefulSetExists(r)
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  dtDNSEntryPoint,
			Value: buildDNSEntryPoint(r.Instance, r.ShortName()),
		})
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		// Reconcile twice since service is created before the stateful set is
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)
		{
			statefulSet := assertStatefulSetExists(r)
			found := 0
			for _, vm := range statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
				if vm.Name == statefulset.InternalProxySecretVolumeName {
					found = found + 1
				}
			}
			assert.Equal(t, 0, found)
		}

		r.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		reconcileAndExpectUpdate(r, true)
		{
			statefulSet := assertStatefulSetExists(r)
			found := 0
			for _, vm := range statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
				if vm.Name == statefulset.InternalProxySecretVolumeName {
					found = found + 1
				}
			}
			assert.Equal(t, 4, found)
		}
	})
	t.Run(`create service`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		reconcileAndExpectUpdate(r, true)
		assertServiceExists(r)

		reconcileAndExpectUpdate(r, true)
		assertStatefulSetExists(r)
	})
	t.Run(`update service`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		setMetricsIngestCapability(r, true)
		reconcileAndExpectUpdate(r, true)
		{
			service := assertServiceExists(r)
			assert.Len(t, service.Spec.Ports, 2)

			assert.Error(t, r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Instance.Namespace}, &appsv1.StatefulSet{}))
		}

		reconcileAndExpectUpdate(r, true)
		{
			service := assertServiceExists(r)
			assert.Len(t, service.Spec.Ports, 2)
			assert.ElementsMatch(t, service.Spec.Ports, []corev1.ServicePort{
				agIngestServicePort, agIngestHttpServicePort,
			})

			statefulSet := assertStatefulSetExists(r)
			assert.Len(t, statefulSet.Spec.Template.Spec.Containers, 1)
		}
		reconcileAndExpectUpdate(r, false)

		setStatsdCapability(r, true)
		reconcileAndExpectUpdate(r, true)
		{
			service := assertServiceExists(r)
			assert.Len(t, service.Spec.Ports, 3)
			assert.ElementsMatch(t, service.Spec.Ports, []corev1.ServicePort{
				agIngestServicePort, agIngestHttpServicePort, statsdIngestServicePort,
			})

			statefulSet := assertStatefulSetExists(r)
			assert.Len(t, statefulSet.Spec.Template.Spec.Containers, 1)
		}

		reconcileAndExpectUpdate(r, true)
		{
			service := assertServiceExists(r)
			assert.ElementsMatch(t, service.Spec.Ports, []corev1.ServicePort{
				agIngestServicePort, agIngestHttpServicePort, statsdIngestServicePort,
			})

			statefulSet := assertStatefulSetExists(r)
			assert.Len(t, statefulSet.Spec.Template.Spec.Containers, 3)
		}
		reconcileAndExpectUpdate(r, false)
		reconcileAndExpectUpdate(r, false)

		setStatsdCapability(r, false)
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, false)
		{
			service := assertServiceExists(r)
			assert.ElementsMatch(t, service.Spec.Ports, []corev1.ServicePort{
				agIngestServicePort, agIngestHttpServicePort,
			})

			statefulSet := assertStatefulSetExists(r)
			assert.Len(t, statefulSet.Spec.Template.Spec.Containers, 1)
		}
	})
}

func TestSetReadinessProbePort(t *testing.T) {
	r := createDefaultReconciler(t)
	stsProps := statefulset.NewStatefulSetProperties(r.Instance, metricsCapability.Properties(), "", "", "", "", "",
		nil, nil, nil,
	)
	sts, err := statefulset.CreateStatefulSet(stsProps)

	assert.NoError(t, err)
	assert.NotNil(t, sts)

	setReadinessProbePort()(sts)

	assert.NotEmpty(t, sts.Spec.Template.Spec.Containers)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port)
	assert.Equal(t, capability.HttpsServicePortName, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.String())
}

func TestReconciler_calculateStatefulSetName(t *testing.T) {
	type fields struct {
		Reconciler *statefulset.Reconciler
		Capability *capability.RoutingCapability
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "instance and module names are defined",
			fields: fields{
				Reconciler: &statefulset.Reconciler{
					Instance: &dynatracev1beta1.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name: "instanceName",
						},
					},
				},
				Capability: metricsCapability,
			},
			want: "instanceName-routing",
		},
		{
			name: "empty instance name",
			fields: fields{
				Reconciler: &statefulset.Reconciler{
					Instance: &dynatracev1beta1.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name: "",
						},
					},
				},
				Capability: metricsCapability,
			},
			want: "-" + metricsCapability.ShortName(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Reconciler: tt.fields.Reconciler,
				Capability: tt.fields.Capability,
			}
			if got := r.calculateStatefulSetName(); got != tt.want {
				t.Errorf("Reconciler.calculateStatefulSetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContainerByName(t *testing.T) {
	verify := func(t *testing.T, containers []corev1.Container, lookingForContainer string, errorMessage string) {
		container, err := getContainerByName(containers, lookingForContainer)
		if errorMessage == "" {
			assert.NoError(t, err)
			assert.NotNil(t, container)
			assert.Equal(t, lookingForContainer, container.Name)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), errorMessage)
			assert.Nil(t, container)
		}
	}

	t.Run("empty slice test cases", func(t *testing.T) {
		verify(t, nil, "", `Cannot find container "" in the provided slice (len 0)`)
		verify(t, []corev1.Container{}, "", `Cannot find container "" in the provided slice (len 0)`)
		verify(t, []corev1.Container{}, "something", `Cannot find container "something" in the provided slice (len 0)`)
	})

	t.Run("non-empty collection but cannot match name", func(t *testing.T) {
		verify(t,
			[]corev1.Container{
				{Name: capability.ActiveGateContainerName},
				{Name: capability.StatsdContainerName},
			},
			capability.EecContainerName,
			fmt.Sprintf(`Cannot find container "%s" in the provided slice (len 2)`, capability.EecContainerName),
		)
	})

	t.Run("happy path", func(t *testing.T) {
		verify(t,
			[]corev1.Container{
				{Name: capability.StatsdContainerName},
			},
			capability.StatsdContainerName,
			"",
		)
	})
}
