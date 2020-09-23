package activegate

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	_ = apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	_ = os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

func TestUpdatePods(t *testing.T) {
	r, instance, err := setupReconciler(t)
	assert.NotNil(t, r)
	assert.NoError(t, err)

	// Check if r is not nil so go linter does not complain
	if r != nil {
		pods, err := r.findOutdatedPods(log.WithName("TestUpdatePods"), instance,
			func(logger logr.Logger, image string, imageID string, secret *corev1.Secret) (bool, error) {
				return imageID == "latest", nil
			})

		assert.NotNil(t, pods)
		assert.NotEmpty(t, pods)
		assert.Equal(t, 1, len(pods))
		assert.Nil(t, err)
	} else {
		assert.Fail(t, "r is nil")
	}
}

func setupReconciler(t *testing.T) (*ReconcileActiveGate, *dynatracev1alpha1.ActiveGate, error) {
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		NewSecret(_const.ActivegateName, _const.DynatraceNamespace, map[string]string{_const.DynatraceApiToken: "42", _const.DynatracePaasToken: "84"}),
		&dynatracev1alpha1.ActiveGate{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: _const.DynatraceNamespace,
				Name:      _const.ActivegateName,
			},
			Spec: dynatracev1alpha1.ActiveGateSpec{
				BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
					Image:  "dynatrace/oneagent:latest",
					APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
				},
			},
		},
	)
	r := &ReconcileActiveGate{
		client:       fakeClient,
		dtcBuildFunc: createFakeDTClient,
		scheme:       scheme.Scheme,
	}
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: _const.DynatraceNamespace,
			Name:      _const.ActivegateName,
		},
	}

	instance := &dynatracev1alpha1.ActiveGate{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	assert.NoError(t, err)

	secret, err := r.getTokenSecret(instance)
	assert.NoError(t, err)

	pod1 := r.newPodForCR(instance, secret)
	pod1.Name = "activegate-pod-1"
	pod1.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Image:   "latest",
			ImageID: "latest",
		},
	}

	pod2 := r.newPodForCR(instance, secret)
	pod2.Name = "activegate-pod-2"
	pod2.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Image:   "outdated",
			ImageID: "outdated",
		},
	}

	err = fakeClient.Create(context.TODO(), pod1)
	assert.NoError(t, err)

	err = fakeClient.Create(context.TODO(), pod2)
	assert.NoError(t, err)
	return r, instance, err
}

func createFakeDTClient(client.Client, *dynatracev1alpha1.ActiveGate, *corev1.Secret) (dtclient.Client, error) {
	dtMockClient := &dtclient.MockDynatraceClient{}
	dtMockClient.On("GetTenantInfo").Return(&dtclient.TenantInfo{}, nil)
	dtMockClient.On("QueryActiveGates", &dtclient.ActiveGateQuery{Hostname: "", NetworkAddress: "", NetworkZone: "default", UpdateStatus: ""}).Return([]dtclient.ActiveGate{}, nil)
	return dtMockClient, nil
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
