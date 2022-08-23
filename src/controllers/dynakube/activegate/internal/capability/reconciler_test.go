package capability

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testName      = "test-name"
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

type testBaseReconciler struct {
	client.Client
	statefulsetReconciler
	mock.Mock
}

func (r *testBaseReconciler) AddOnAfterStatefulSetCreateListener(_ kubeobjects.StatefulSetEvent) {}

func (r *testBaseReconciler) Reconcile() (update bool, err error) {
	args := r.Called()
	return args.Bool(0), args.Error(1)
}

func (r *testBaseReconciler) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return r.Client.Get(ctx, key, obj)
}

type PseudoReconcilerMock struct {
	mock.Mock
}

var _ kubeobjects.PseudoReconciler = (*PseudoReconcilerMock)(nil)

func (r *PseudoReconcilerMock) Reconcile() (update bool, err error) {
	args := r.Called()
	return args.Get(0).(bool), args.Error(1)
}

func TestNewReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) (*Reconciler, *testBaseReconciler) {
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
	baseReconciler := &testBaseReconciler{
		Client: clt,
	}

	customPropertiesReconcilerMock := &PseudoReconcilerMock{}
	customPropertiesReconcilerMock.On("Reconcile").Return(false, nil)

	r := NewReconciler(clt, metricsCapability, instance, baseReconciler, customPropertiesReconcilerMock)
	require.NotNil(t, r)
	require.NotNil(t, r.statefulsetReconciler)
	require.NotNil(t, r.Dynakube)
	require.NotEmpty(t, r.Dynakube.ObjectMeta.Name)

	return r, baseReconciler
}

func TestReconcile(t *testing.T) {
	assertStatefulSetExists := func(r *Reconciler) *appsv1.StatefulSet {
		statefulSet := new(appsv1.StatefulSet)
		assert.NoError(t, r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Dynakube.Namespace}, statefulSet))
		assert.NotNil(t, statefulSet)
		return statefulSet
	}
	assertServiceExists := func(r *Reconciler) *corev1.Service {
		svc := new(corev1.Service)
		assert.NoError(t, r.Get(context.TODO(), client.ObjectKey{Name: capability.BuildServiceName(r.Dynakube.Name, r.ShortName()), Namespace: r.Dynakube.Namespace}, svc))
		assert.NotNil(t, svc)
		return svc
	}
	reconcileAndExpectUpdate := func(r *Reconciler, updateExpected bool) {
		update, err := r.Reconcile()
		assert.NoError(t, err)
		assert.Equal(t, updateExpected, update)
	}
	setStatsdCapability := func(r *Reconciler, wantEnabled bool) {
		kubeobjects.SwitchCapability(r.Dynakube, dynatracev1beta1.StatsdIngestCapability, wantEnabled)
	}
	setMetricsIngestCapability := func(r *Reconciler, wantEnabled bool) {
		kubeobjects.SwitchCapability(r.Dynakube, dynatracev1beta1.MetricsIngestCapability, wantEnabled)
	}

	agIngestServicePort := corev1.ServicePort{
		Name:       consts.HttpsServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HttpsServicePort,
		TargetPort: intstr.FromString(consts.HttpsServicePortName),
	}
	agIngestHttpServicePort := corev1.ServicePort{
		Name:       consts.HttpServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HttpServicePort,
		TargetPort: intstr.FromString(consts.HttpServicePortName),
	}
	statsdIngestServicePort := corev1.ServicePort{
		Name:       consts.StatsdIngestPortName,
		Protocol:   corev1.ProtocolUDP,
		Port:       consts.StatsdIngestPort,
		TargetPort: intstr.FromString(consts.StatsdIngestTargetPort),
	}

	t.Run(`reconcile custom properties`, func(t *testing.T) {
		r, baseReconciler := createDefaultReconciler(t)

		metricsCapability.Properties().CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}

		baseReconciler.On("Reconcile").Return(true, nil).Run(func(args mock.Arguments) {
			err := r.Create(context.TODO(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.Dynakube.Name + "-" + metricsCapability.ShortName() + "-" + customproperties.Suffix,
					Namespace: r.Dynakube.Namespace,
				},
				Data: map[string][]byte{customproperties.DataKey: []byte(testValue)},
			})
			require.NoError(t, err)
		})

		// Reconcile twice since service is created before the stateful set is
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)

		var customProperties corev1.Secret
		err := r.Get(context.TODO(), client.ObjectKey{Name: r.Dynakube.Name + "-" + metricsCapability.ShortName() + "-" + customproperties.Suffix, Namespace: r.Dynakube.Namespace}, &customProperties)
		assert.NoError(t, err)
		assert.NotNil(t, customProperties)
		assert.Contains(t, customProperties.Data, customproperties.DataKey)
		assert.Equal(t, testValue, string(customProperties.Data[customproperties.DataKey]))
	})
	t.Run(`create stateful set`, func(t *testing.T) {
		r, baseReconciler := createDefaultReconciler(t)

		baseReconciler.On("Reconcile").Return(true, nil).Run(func(args mock.Arguments) {
			err := r.Create(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{{Name: dtDnsEntryPoint, Value: buildDNSEntryPoint(r.Dynakube, r.ShortName())}}}},
						},
					},
				},
			})
			require.NoError(t, err)
		})

		// Reconcile twice since service is created before the stateful set is
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)

		statefulSet := assertStatefulSetExists(r)
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  dtDnsEntryPoint,
			Value: buildDNSEntryPoint(r.Dynakube, r.ShortName()),
		})
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r, baseReconciler := createDefaultReconciler(t)

		call := baseReconciler.On("Reconcile").Return(true, nil).Run(func(args mock.Arguments) {
			err := r.Create(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{}},
						},
					},
				},
			})
			require.NoError(t, err)
		})

		// Reconcile twice since service is created before the stateful set is
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)
		{
			statefulSet := assertStatefulSetExists(r)
			assert.Equal(t, 0, len(statefulSet.Spec.Template.Spec.Containers[0].Env))
		}

		r.Dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}

		call.Run(func(args mock.Arguments) {
			err := r.Update(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Env: []corev1.EnvVar{
									{Name: "a", Value: "a"},
									{Name: "b", Value: "b"},
									{Name: "c", Value: "c"},
									{Name: "d", Value: "d"},
								},
							}},
						},
					},
				},
			})
			require.NoError(t, err)
		})

		reconcileAndExpectUpdate(r, true)
		{
			statefulSet := assertStatefulSetExists(r)
			assert.Equal(t, 4, len(statefulSet.Spec.Template.Spec.Containers[0].Env))
		}
	})
	t.Run(`create service`, func(t *testing.T) {
		r, baseReconciler := createDefaultReconciler(t)

		call := baseReconciler.On("Reconcile").Return(true, nil)

		reconcileAndExpectUpdate(r, true)
		assertServiceExists(r)

		call.Run(func(args mock.Arguments) {
			err := r.Create(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
			})
			require.NoError(t, err)
		})

		reconcileAndExpectUpdate(r, true)
		assertStatefulSetExists(r)
	})
	t.Run(`update service`, func(t *testing.T) {
		r, baseReconciler := createDefaultReconciler(t)

		call := baseReconciler.On("Reconcile").Return(true, nil)

		setMetricsIngestCapability(r, true)
		reconcileAndExpectUpdate(r, true)
		{
			service := assertServiceExists(r)
			assert.Len(t, service.Spec.Ports, 2)

			assert.Error(t, r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Dynakube.Namespace}, &appsv1.StatefulSet{}))
		}

		call.Run(func(args mock.Arguments) {
			err := r.Create(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{}},
						},
					},
				},
			})
			require.NoError(t, err)
		})

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

		call.Return(false, nil).Run(func(args mock.Arguments) {
			err := r.Update(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{}},
						},
					},
				},
			})
			require.NoError(t, err)
		})
		reconcileAndExpectUpdate(r, false)

		call.Return(true, nil)
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

		call.Run(func(args mock.Arguments) {
			err := r.Update(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{}, {}, {}},
						},
					},
				},
			})
			require.NoError(t, err)
		})

		reconcileAndExpectUpdate(r, true)
		{
			service := assertServiceExists(r)
			assert.ElementsMatch(t, service.Spec.Ports, []corev1.ServicePort{
				agIngestServicePort, agIngestHttpServicePort, statsdIngestServicePort,
			})

			statefulSet := assertStatefulSetExists(r)
			assert.Len(t, statefulSet.Spec.Template.Spec.Containers, 3)
		}
		call.Return(false, nil)
		reconcileAndExpectUpdate(r, false)
		reconcileAndExpectUpdate(r, false)

		setStatsdCapability(r, false)
		call.Return(true, nil)
		reconcileAndExpectUpdate(r, true)
		reconcileAndExpectUpdate(r, true)
		call.Return(false, nil)

		call.Run(func(args mock.Arguments) {
			err := r.Update(context.TODO(), &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.calculateStatefulSetName(),
					Namespace: r.Dynakube.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{}},
						},
					},
				},
			})
			require.NoError(t, err)
		})

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

func TestReconciler_calculateStatefulSetName(t *testing.T) {
	type fields struct {
		Instance   *dynatracev1beta1.DynaKube
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
				Instance: &dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instanceName",
					},
				},
				Capability: metricsCapability,
			},
			want: "instanceName-routing",
		},
		{
			name: "empty instance name",
			fields: fields{
				Instance: &dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: "",
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
				Dynakube:   tt.fields.Instance,
				Capability: tt.fields.Capability,
			}
			if got := r.calculateStatefulSetName(); got != tt.want {
				t.Errorf("PseudoReconciler.calculateStatefulSetName() = %v, want %v", got, tt.want)
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
				{Name: consts.ActiveGateContainerName},
				{Name: consts.StatsdContainerName},
			},
			consts.EecContainerName,
			fmt.Sprintf(`Cannot find container "%s" in the provided slice (len 2)`, consts.EecContainerName),
		)
	})

	t.Run("happy path", func(t *testing.T) {
		verify(t,
			[]corev1.Container{
				{Name: consts.StatsdContainerName},
			},
			consts.StatsdContainerName,
			"",
		)
	})
}
