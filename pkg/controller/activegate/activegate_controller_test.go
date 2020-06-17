package activegate

import (
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestUpdatePods(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		NewSecret("activegate", "dynatrace", map[string]string{_const.DynatraceApiToken: "42", _const.DynatracePaasToken: "84"}),
	)
	r := ReconcileActiveGate{
		client: fakeClient,
	}
	request := reconcile.Request{}

	reconciliation, _ := r.Reconcile(request)

	assert.NotNil(t, reconciliation)
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
