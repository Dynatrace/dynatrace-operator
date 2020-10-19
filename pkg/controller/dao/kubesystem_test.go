package dao

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/factory"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	_ = apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	_ = os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

func TestFindKubeSystemUID(t *testing.T) {
	t.Run("FindKubeSystemUID", func(t *testing.T) {
		fakeClient := factory.CreateFakeClient()

		kubeSystemNamespace := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				UID:  factory.KubeSystemUID,
				Name: _const.KubeSystemNamespace,
			},
		}

		err := fakeClient.Update(context.TODO(), &kubeSystemNamespace)
		assert.NoError(t, err)

		uid, err := FindKubeSystemUID(fakeClient)
		assert.NoError(t, err)
		assert.NotNil(t, uid)
		assert.Equal(t, types.UID(factory.KubeSystemUID), uid)
	})
	t.Run("FindKubeSystemUID no namespace", func(t *testing.T) {
		fakeClient := factory.CreateFakeClient()
		kubeSystemNamespace := &v1.Namespace{}

		err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: _const.KubeSystemNamespace}, kubeSystemNamespace)
		assert.NoError(t, err)

		err = fakeClient.Delete(context.TODO(), kubeSystemNamespace)
		assert.NoError(t, err)

		uid, err := FindKubeSystemUID(fakeClient)

		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
		assert.NotNil(t, uid)
		assert.Empty(t, uid)
	})
}
