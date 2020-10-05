//+build integration

package activegate

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

func TestUpdatePods_RemoteRepository(t *testing.T) {
	r, _, err := setupReconciler(t, &activeGateUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	reconciliation, err := r.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: _const.DynatraceNamespace,
			Name:      _const.ActivegateName,
		}})

	assert.NotNil(t, reconciliation)
	assert.NoError(t, err)
}
