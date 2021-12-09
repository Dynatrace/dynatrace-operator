package capability

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	rsfs "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		}}
	r := NewReconciler(metricsCapability, clt, clt, scheme.Scheme, instance)
	require.NotNil(t, r)
	require.NotNil(t, r.Client)
	require.NotNil(t, r.Instance)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile custom properties`, func(t *testing.T) {
		r := createDefaultReconciler(t)

		metricsCapability.Properties().CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}
		_, err := r.Reconcile()
		assert.NoError(t, err)

		// Reconcile twice since service is created before the stateful set is
		_, err = r.Reconcile()
		assert.NoError(t, err)

		var customProperties corev1.Secret
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + metricsCapability.ShortName() + "-" + customproperties.Suffix, Namespace: r.Instance.Namespace}, &customProperties)
		assert.NoError(t, err)
		assert.NotNil(t, customProperties)
		assert.Contains(t, customProperties.Data, customproperties.DataKey)
		assert.Equal(t, testValue, string(customProperties.Data[customproperties.DataKey]))
	})
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		// Reconcile twice since service is created before the stateful set is
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  dtDNSEntryPoint,
			Value: buildDNSEntryPoint(r.Instance, r.ShortName()),
		})
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		// Reconcile twice since service is created before the stateful set is
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		r.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Instance.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := false
		for _, env := range newStatefulSet.Spec.Template.Spec.Containers[0].Env {
			if env.Name == rsfs.DTInternalProxy {
				found = true
				assert.Equal(t, testValue, env.Value)
			}
		}
		assert.True(t, found)
	})
	t.Run(`create service`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()
		assert.True(t, update)
		assert.NoError(t, err)

		svc := &corev1.Service{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: BuildServiceName(r.Instance.Name, r.ShortName()), Namespace: r.Instance.Namespace}, svc)
		assert.NoError(t, err)
		assert.NotNil(t, svc)

		update, err = r.Reconcile()
		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.calculateStatefulSetName(), Namespace: r.Instance.Namespace}, statefulSet)
		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
}

func TestSetReadinessProbePort(t *testing.T) {
	r := createDefaultReconciler(t)
	stsProps := rsfs.NewStatefulSetProperties(r.Instance, metricsCapability.Properties(), "", "", "", "", "", nil, nil, nil)
	sts, err := rsfs.CreateStatefulSet(stsProps)

	assert.NoError(t, err)
	assert.NotNil(t, sts)

	setReadinessProbePort()(sts)

	assert.NotEmpty(t, sts.Spec.Template.Spec.Containers)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port)
	assert.Equal(t, consts.HttpsServiceTargetPort, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.String())
}

func TestReconciler_calculateStatefulSetName(t *testing.T) {
	type fields struct {
		Reconciler *rsfs.Reconciler
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
				Reconciler: &rsfs.Reconciler{
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
				Reconciler: &rsfs.Reconciler{
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
