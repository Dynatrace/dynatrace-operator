// +build integration

package integrationtests

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReconcileOneAgent_ReconcileOnEmptyEnvironment(t *testing.T) {
	oaName := "dynakube"

	e, err := newTestEnvironment()
	assert.NoError(t, err, "failed to start test environment")

	defer e.Stop()

	e.AddOneAgent(oaName, &dynatracev1alpha1.DynaKubeSpec{
		APIURL: DefaultTestAPIURL,
		Tokens: "token-test",
		ClassicFullStack: dynatracev1alpha1.FullStackSpec{
			Enabled: true,
		},
	})

	_, err = e.Reconciler.Reconcile(context.TODO(), newReconciliationRequest(oaName))
	assert.NoError(t, err, "error reconciling")

	// Check if deamonset has been created and has correct namespace and name.
	dsActual := &appsv1.DaemonSet{}

	err = e.Client.Get(context.TODO(), types.NamespacedName{Name: oaName + "-" + oneagent.ClassicFeature, Namespace: DefaultTestNamespace}, dsActual)
	assert.NoError(t, err, "failed to get daemonset")

	assert.Equal(t, DefaultTestNamespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, oaName+"-"+oneagent.ClassicFeature, dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "DNS policy should ClusterFirst by default")
}
