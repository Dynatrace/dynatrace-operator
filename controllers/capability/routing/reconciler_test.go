package routing

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
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

func TestNewReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) *Reconciler {
	log := logger.NewDTLogger()
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			},
		}).
		Build()
	dtc := &dtclient.MockDynatraceClient{}
	instance := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		}}
	imgVerProvider := func(img string, dockerConfig *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
		return dtversion.ImageVersion{}, nil
	}

	r := NewReconciler(clt, clt, scheme.Scheme, dtc, log, instance, imgVerProvider)
	require.NotNil(t, r)
	require.NotNil(t, r.Client)
	require.NotNil(t, r.Instance)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile custom properties`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		r.Instance.Spec.RoutingSpec.CapabilityProperties.CustomProperties = &v1alpha1.DynaKubeValueSource{
			Value: testValue,
		}
		_, err := r.Reconcile()
		assert.NoError(t, err)

		// Reconcile twice since service is created before the stateful set is
		_, err = r.Reconcile()
		assert.NoError(t, err)

		var customProperties corev1.Secret
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + Module + "-" + customproperties.Suffix, Namespace: r.Instance.Namespace}, &customProperties)
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
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + StatefulSetSuffix, Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  DTDNSEntryPoint,
			Value: buildDNSEntryPoint(r.Instance),
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
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + StatefulSetSuffix, Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		r.Instance.Spec.Proxy = &v1alpha1.DynaKubeProxy{Value: testValue}
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + StatefulSetSuffix, Namespace: r.Instance.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := false
		for _, env := range newStatefulSet.Spec.Template.Spec.Containers[0].Env {
			if env.Name == capability.DTInternalProxy {
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

		service := &corev1.Service{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: BuildServiceName(r.Instance.Name, Module), Namespace: r.Instance.Namespace}, service)
		assert.NoError(t, err)
		assert.NotNil(t, service)

		update, err = r.Reconcile()
		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + StatefulSetSuffix, Namespace: r.Instance.Namespace}, statefulSet)
		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
}

func TestSetReadinessProbePort(t *testing.T) {
	r := createDefaultReconciler(t)
	stsProps := capability.NewStatefulSetProperties(r.Instance, &r.Instance.Spec.RoutingSpec.CapabilityProperties, "", "", "", "", "")
	sts, err := capability.CreateStatefulSet(stsProps)

	assert.NoError(t, err)
	assert.NotNil(t, sts)

	setReadinessProbePort(r.Instance)(sts)

	assert.NotEmpty(t, sts.Spec.Template.Spec.Containers)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet)
	assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port)
	assert.Equal(t, serviceTargetPort, sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.String())
}
