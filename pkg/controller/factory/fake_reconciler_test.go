package factory

import (
	"context"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func init() {
	_ = apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	_ = os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

func TestCreateFakeClient(t *testing.T) {
	fakeClient := CreateFakeClient()
	assert.NotNil(t, fakeClient)

	secret := &corev1.Secret{}
	activeGate := &v1alpha1.ActiveGate{}

	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: _const.ActivegateName, Namespace: _const.DynatraceNamespace}, secret)
	assert.NoError(t, err)
	assert.NotNil(t, secret)

	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: _const.ActivegateName, Namespace: _const.DynatraceNamespace}, activeGate)
	assert.NoError(t, err)
	assert.NotNil(t, activeGate)
}
