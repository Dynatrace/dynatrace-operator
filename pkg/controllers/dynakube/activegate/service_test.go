package activegate

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		clt := fake.NewClient()
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
