package namespace

import (
	"context"
	_ "embed"
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//go:embed init.sh.test-sample
var scriptSample string

func TestReconcileNamespace(t *testing.T) {
	c := fake.NewClient(
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: "https://test-url/api",
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
				InfraMonitoring: dynatracev1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				EnvironmentID: "abc12345",
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{
						"node1": {},
					},
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-namespace",
				Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
				UID:  "42",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
		},
	)

	r := ReconcileNamespaces{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
		namespace: "dynatrace",
	}

	_, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
	assert.NoError(t, err)

	var nsSecret corev1.Secret
	require.NoError(t, c.Get(context.TODO(), client.ObjectKey{
		Name:      "dynatrace-dynakube-config",
		Namespace: "test-namespace",
	}, &nsSecret))

	require.Len(t, nsSecret.Data, 1)
	require.Contains(t, nsSecret.Data, "init.sh")
	require.NotEmpty(t, scriptSample) // sanity check to confirm that the sample script has been embedded
	require.Equal(t, scriptSample, string(nsSecret.Data["init.sh"]))
}
