//+build integration

package activegate

import (
	"context"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func init() {
	apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")
}

func TestUpdatePods(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		NewSecret("activegate", "dynatrace", map[string]string{_const.DynatraceApiToken: "42", _const.DynatracePaasToken: "84"}),
		&dynatracev1alpha1.ActiveGate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "activegate",
				Namespace: "dynatrace",
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
			Namespace: "dynatrace",
			Name:      "activegate",
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
		corev1.ContainerStatus{
			Image:   "dynatrace/oneagent:0.8",
			ImageID: "docker-pullable://dynatrace/oneagent@sha256:125422510b677dfa19bf1041f3f2f3592b0ca683c92071f68940607c418bbdd9",
		},
	}

	pod2 := r.newPodForCR(instance, secret)
	pod2.Name = "activegate-pod-2"
	pod2.Status.ContainerStatuses = []corev1.ContainerStatus{
		corev1.ContainerStatus{
			Image:   "dynatrace/oneagent:0.7",
			ImageID: "docker-pullable://dynatrace/oneagent@sha256:e2050c728872b1f4cabd50546a696c3d33ebcfb9ed49528b4e317d5c69a6ef05",
		},
	}

	err = fakeClient.Create(context.TODO(), pod1)
	assert.NoError(t, err)

	err = fakeClient.Create(context.TODO(), pod2)
	assert.NoError(t, err)

	reconciliation, err := r.Reconcile(request)

	assert.NotNil(t, reconciliation)
	assert.Nil(t, err)
}

func createFakeDTClient(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
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
