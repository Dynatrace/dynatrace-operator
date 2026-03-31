package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testComponentFeature = "test-component-feature"
	testAPIURL           = "https://demo.dev.dynatracelabs.com/api"
)

func createTestDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
		},
	}
}

func TestCreateService(t *testing.T) {
	agHTTPSPort := corev1.ServicePort{
		Name:       consts.HTTPSServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HTTPSServicePort,
		TargetPort: intstr.FromString(consts.HTTPSServicePortName),
	}
	agHTTPPort := corev1.ServicePort{
		Name:       consts.HTTPServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HTTPServicePort,
		TargetPort: intstr.FromString(consts.HTTPServicePortName),
	}

	t.Run("check service name, labels and selector", func(t *testing.T) {
		dk := createTestDynaKube()
		service := CreateService(dk)

		assert.NotNil(t, service)
		assert.Equal(t, dk.Name+"-"+consts.MultiActiveGateName, service.Name)
		assert.Equal(t, dk.Namespace, service.Namespace)

		expectedLabels := map[string]string{
			k8slabel.AppCreatedByLabel: testName,
			k8slabel.AppComponentLabel: k8slabel.ActiveGateComponentLabel,
			k8slabel.AppNameLabel:      version.AppName,
			k8slabel.AppVersionLabel:   version.Version,
		}
		assert.Equal(t, expectedLabels, service.Labels)

		expectedSelector := map[string]string{
			k8slabel.AppCreatedByLabel: testName,
			k8slabel.AppManagedByLabel: version.AppName,
			k8slabel.AppNameLabel:      k8slabel.ActiveGateComponentLabel,
		}
		serviceSpec := service.Spec
		assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
		assert.Equal(t, expectedSelector, serviceSpec.Selector)

		ports := service.Spec.Ports
		assert.Contains(t, ports, agHTTPSPort)
		assert.Contains(t, ports, agHTTPPort)
	})
}

func TestCreateOrUpdateService(t *testing.T) {
	dk := createTestDynaKube()
	dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
		activegate.RoutingCapability.DisplayName,
		activegate.KubeMonCapability.DisplayName,
		activegate.MetricsIngestCapability.DisplayName,
		activegate.DynatraceAPICapability.DisplayName,
	}

	getService := func(t *testing.T, clt client.Client) *corev1.Service {
		t.Helper()
		service := &corev1.Service{}
		err := clt.Get(t.Context(), client.ObjectKey{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		return service
	}

	tests := []struct {
		name   string
		mutate func(*corev1.Service)
	}{
		{
			"ports get updated",
			func(svc *corev1.Service) {
				svc.Spec.Ports = []corev1.ServicePort{}
			},
		},
		{
			"labels get updated",
			func(svc *corev1.Service) {
				svc.Labels = map[string]string{}
			},
		},
		{
			"selector gets updated",
			func(svc *corev1.Service) {
				svc.Spec.Selector = map[string]string{}
			},
		},
	}

	for _, test := range tests {
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		r := &Reconciler{client: clt}

		err := r.createOrUpdateService(t.Context(), dk)
		require.NoError(t, err)

		service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}}
		result, err := controllerutil.CreateOrUpdate(t.Context(), clt, service, func() error {
			test.mutate(service)

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, controllerutil.OperationResultUpdated, result)

		err = r.createOrUpdateService(t.Context(), dk)
		require.NoError(t, err)

		actualService := getService(t, clt)
		desiredService := CreateService(dk)
		assert.Equal(t, desiredService.Labels, actualService.Labels)
		assert.Equal(t, desiredService.Spec, actualService.Spec)
		assert.NotEqual(t, actualService, service)
	}
}

func TestSetAGServiceIPs(t *testing.T) {
	buildDynakube := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				EnableIstio: true,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
				},
			}}
	}

	t.Run("sets ServiceIPs from existing service ClusterIPs", func(t *testing.T) {
		dk := buildDynakube()
		expectedIPs := []string{"10.0.0.1", "fd00::1"}

		svc := CreateService(dk)
		svc.Spec.ClusterIPs = expectedIPs

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(svc).
			Build()

		r := &Reconciler{client: clt}

		err := r.setAGServiceIPs(t.Context(), dk)
		require.NoError(t, err)
		assert.Equal(t, expectedIPs, dk.Status.ActiveGate.ServiceIPs)
	})

	t.Run("returns error when service does not exist", func(t *testing.T) {
		dk := buildDynakube()

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			Build()

		r := &Reconciler{client: clt}

		err := r.setAGServiceIPs(t.Context(), dk)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("retry if not there", func(t *testing.T) {
		dk := buildDynakube()
		expectedIPs := []string{"10.0.0.1", "fd00::1"}

		svc := CreateService(dk)
		svc.Spec.ClusterIPs = expectedIPs
		expectedAttempts := 2
		attemptCounter := 0

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if attemptCounter < expectedAttempts {
						attemptCounter++

						return k8serrors.NewNotFound(schema.GroupResource{}, "test")
					}
					svc.DeepCopyInto(obj.(*corev1.Service))

					return nil
				},
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return errors.New("UNEXPECTED")
				},
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					return errors.New("UNEXPECTED")
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					return errors.New("UNEXPECTED")
				},
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					return errors.New("UNEXPECTED")
				},
			}).
			Build()

		r := &Reconciler{client: clt}

		err := r.setAGServiceIPs(t.Context(), dk)
		require.NoError(t, err)
		assert.Equal(t, expectedIPs, dk.Status.ActiveGate.ServiceIPs)
		assert.Equal(t, expectedAttempts, attemptCounter)
	})

	t.Run("clears ServiceIPs when service has no ClusterIPs", func(t *testing.T) {
		dk := buildDynakube()
		dk.Status.ActiveGate.ServiceIPs = []string{"10.0.0.1"}

		svc := CreateService(dk)
		// ClusterIPs intentionally left empty

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(svc).
			Build()

		r := &Reconciler{client: clt}

		err := r.setAGServiceIPs(t.Context(), dk)
		require.NoError(t, err)
		assert.Empty(t, dk.Status.ActiveGate.ServiceIPs)
	})
}
