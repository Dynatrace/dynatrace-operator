package namespace_mapping

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//go:embed init.sh.test-sample
var scriptSample string

var (
	testNamespace1 = "namespace1"
	testNamespace2 = "namespace2"
	testDynaKube1  = "dynakube1"
	testDynaKube2  = "dynakube2"
	testApiUrl     = "https://test-url/api"
)

func TestReconcileNamespaceMapping_EmptyConfigMap(t *testing.T) {
	c := fake.NewClient()
	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	_, err := r.Reconcile(context.TODO(), reconcile.Request{})
	assert.NoError(t, err)
}

func TestReconcileNamespaceMapping_TwoDynakubes(t *testing.T) {
	c := fake.NewClient(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
				UID:  "42",
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceMappingConfigMap},
			Data: map[string]string{
				testNamespace1: testDynaKube1,
				testNamespace2: testDynaKube2,
			},
		},
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynaKube1},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				InfraMonitoring: dynatracev1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
					TenantUUID: "abc12345",
				},
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{
						"node1": {},
					},
				},
			},
		},
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynaKube2},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				Tokens: "secret2",
				InfraMonitoring: dynatracev1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
					TenantUUID: "abc12345",
				},
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{
						"node2": {},
					},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testDynaKube1},
			Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret2"},
			Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
		})

	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	_, err := r.Reconcile(context.TODO(), reconcile.Request{})
	assert.NoError(t, err)

	var initSecret1 corev1.Secret
	require.NoError(t, c.Get(context.TODO(), client.ObjectKey{
		Name:      "dynatrace-dynakube-config",
		Namespace: testNamespace1,
	}, &initSecret1))

	require.Len(t, initSecret1.Data, 1)
	require.Contains(t, initSecret1.Data, "init.sh")
	require.NotEmpty(t, scriptSample) // sanity check to confirm that the sample script has been embedded
	require.Equal(t, scriptSample, string(initSecret1.Data["init.sh"]))

	var initSecret2 corev1.Secret
	require.NoError(t, c.Get(context.TODO(), client.ObjectKey{
		Name:      "dynatrace-dynakube-config",
		Namespace: testNamespace2,
	}, &initSecret2))

	require.Len(t, initSecret2.Data, 1)
	require.Contains(t, initSecret2.Data, "init.sh")
	require.NotEmpty(t, scriptSample) // sanity check to confirm that the sample script has been embedded
	require.Equal(t, scriptSample, string(initSecret2.Data["init.sh"]))
}
