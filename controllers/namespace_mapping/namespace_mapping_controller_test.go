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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//go:embed init.sh.test-sample
var scriptSample string

var (
	testNamespace1      = "namespace1"
	testNamespace2      = "namespace2"
	testDynaKubeName1   = "dynakube1"
	testDynaKubeName2   = "dynakube2"
	testApiUrl          = "https://test-url/api"
	testProxy           = "testproxy.com"
	testtrustCAsCM      = "testtrustedCAsConfigMap"
	testCAValue         = "somecertificate"
	testTenantUUID      = "abc12345"
	kubesystemNamespace = "kube-system"
	kubesystemUID       = types.UID("42")

	testdk1 = &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynaKubeName1},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: testApiUrl,
			InfraMonitoring: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{
					"node1": {},
				},
			},
		},
	}

	testdk2 = &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynaKubeName2},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: testApiUrl,
			Tokens: "secret2",
			InfraMonitoring: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{
					"node2": {},
				},
			},
		},
	}

	testSecretDk1 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testDynaKubeName1},
		Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
	}

	testSecretDk2 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2"},
		Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
	}
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
				Name: kubesystemNamespace,
				UID:  kubesystemUID,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceMappingConfigMap},
			Data: map[string]string{
				testNamespace1: testDynaKubeName1,
				testNamespace2: testDynaKubeName2,
			},
		},
		testdk1,
		testdk2,
		testSecretDk1,
		testSecretDk2,
	)

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

func TestCodeModulesNamespaceMapping_SingleData(t *testing.T) {
	testdata := map[string]string{
		testNamespace1: testDynaKubeName1,
	}

	expectedMap := []namespaceMapping{
		{
			namespace: testNamespace1,
			dynakube:  testDynaKubeName1,
		},
	}

	mapping := getCodeModulesNamespaceMapping(testdata)
	assert.Equal(t, expectedMap, mapping)
}

func TestCodeModulesNamespaceMapping_NoData(t *testing.T) {
	expectedMap := []namespaceMapping{
		{
			namespace: testNamespace1,
			dynakube:  testDynaKubeName1,
		},
	}

	mapping := getCodeModulesNamespaceMapping(nil)
	assert.Nil(t, mapping)
	assert.NotEqual(t, expectedMap, mapping)
}

func TestCodeModulesNamespaceMapping_JustNamespace(t *testing.T) {
	testdata := map[string]string{
		testNamespace1: "",
	}

	expectedMap := []namespaceMapping{
		{
			namespace: testNamespace1,
			dynakube:  "",
		},
	}

	mapping := getCodeModulesNamespaceMapping(testdata)
	assert.Equal(t, expectedMap, mapping)
}

func TestGetInfraMonitoringHostNodes_NoNodes(t *testing.T) {
	c := fake.NewClient()
	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	expectedNodes := map[string]string{}

	imNodes, err := r.getInfraMonitoringNodes()
	assert.Equal(t, expectedNodes, imNodes)
	assert.NoError(t, err)
}

func TestGetInfraMonitoringHostNodes_WithNodes(t *testing.T) {
	c := fake.NewClient(testdk1)

	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	expectedNodes := map[string]string{
		"node1": testTenantUUID,
	}

	imNodes, err := r.getInfraMonitoringNodes()
	assert.Equal(t, expectedNodes, imNodes)
	assert.NoError(t, err)
}

func TestPrepareScriptForDynaKube_NoDynakube(t *testing.T) {
	c := fake.NewClient()
	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	imNodes := map[string]string{
		"node1": testTenantUUID,
	}

	s, err := r.prepareScriptForDynaKube("", kubesystemUID, imNodes)
	assert.Error(t, err, "dynakubes.dynatrace.com \"\" not found")
	assert.Nil(t, s)
}

func TestPrepareScriptForDynaKube_FullData(t *testing.T) {
	c := fake.NewClient(testdk1, testSecretDk1)
	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	imNodes := map[string]string{
		"node1": testTenantUUID,
	}

	expectedScript := &script{
		ApiUrl:        testApiUrl,
		SkipCertCheck: false,
		PaaSToken:     "42",
		Proxy:         "",
		TrustedCAs:    nil,
		ClusterID:     string(kubesystemUID),
		TenantUUID:    testTenantUUID,
		IMNodes:       imNodes,
	}

	s, err := r.prepareScriptForDynaKube(testDynaKubeName1, kubesystemUID, imNodes)
	assert.NoError(t, err)
	assert.Equal(t, expectedScript, s)
}

func TestPrepareScriptForDynaKube_FullData_WithProxyAndCerts(t *testing.T) {
	c := fake.NewClient(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testtrustCAsCM},
			Data: map[string]string{
				"certs": testCAValue,
			},
		},
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynaKubeName1},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				Proxy: &dynatracev1alpha1.DynaKubeProxy{
					Value: testProxy,
				},
				TrustedCAs: testtrustCAsCM,
				InfraMonitoring: dynatracev1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
					TenantUUID: testTenantUUID,
				},
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{
						"node1": {},
					},
				},
			},
		},
		testSecretDk1)

	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	imNodes := map[string]string{
		"node1": testTenantUUID,
	}

	expectedScript := &script{
		ApiUrl:        testApiUrl,
		SkipCertCheck: false,
		PaaSToken:     "42",
		Proxy:         testProxy,
		TrustedCAs:    []byte(testCAValue),
		ClusterID:     string(kubesystemUID),
		TenantUUID:    testTenantUUID,
		IMNodes:       imNodes,
	}

	s, err := r.prepareScriptForDynaKube(testDynaKubeName1, kubesystemUID, imNodes)
	assert.NoError(t, err)
	assert.Equal(t, expectedScript, s)
}

func TestReplicateInitScriptAsSecret(t *testing.T) {
	c := fake.NewClient(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystemNamespace,
				UID:  kubesystemUID,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceMappingConfigMap},
			Data: map[string]string{
				testNamespace1: testDynaKubeName1,
				testNamespace2: testDynaKubeName2,
			},
		},
		testdk1,
		testdk2,
		testSecretDk1,
		testSecretDk2)

	r := &ReconcileNamespaceMapping{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
	}

	mapping := []namespaceMapping{
		{
			namespace: testNamespace1,
			dynakube:  testDynaKubeName1,
		},
		{
			namespace: testNamespace2,
			dynakube:  testDynaKubeName2,
		},
	}

	imNodes := map[string]string{
		"node1": testTenantUUID,
		"node2": testTenantUUID,
	}

	err := r.replicateInitScriptAsSecret(mapping, kubesystemUID, imNodes)
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
