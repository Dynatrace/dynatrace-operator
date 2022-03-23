package initgeneration

import (
	"context"
	_ "embed"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	operatorNamespace        = "dynatrace"
	testNamespaceName        = "namespace"
	testOtherNamespaceName   = "other-namespace"
	testDynakubeComplexName  = "dynakubeComplex"
	testDynakubeSimpleName   = "dynakubeSimple"
	testTokensName           = "kitchen-sink"
	testApiUrl               = "https://test-url/api"
	testProxy                = "testproxy.com"
	testClusterCa            = "testClusterCaConfigMap"
	testCAValue              = "somecertificate"
	testTenantUUID           = "abc12345"
	kubesystemNamespace      = "kube-system"
	kubesystemUID            = types.UID("42")
	testNode1Name            = "node1"
	testNode2Name            = "node2"
	testNodeWithSelectorName = "nodeWselector"
	testSelectorLabels       = map[string]string{"test": "label"}

	testDynakubeComplex = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeComplexName, Namespace: operatorNamespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:    testApiUrl,
			Proxy:     &dynatracev1beta1.DynaKubeProxy{Value: testProxy},
			ClusterCa: testClusterCa,
			Tokens:    testTokensName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{
						Args: []string{
							"--something=else",
							"",
						},
					},
				}},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
				},
				TlsSecretName: "testing",
			},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1beta1.OneAgentStatus{
				Instances: map[string]dynatracev1beta1.OneAgentInstance{
					testNode1Name: {},
				},
			},
		},
	}

	testDynakubeSimple = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeSimpleName, Namespace: operatorNamespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:   testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{}},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1beta1.OneAgentStatus{
				Instances: map[string]dynatracev1beta1.OneAgentInstance{
					testNode2Name: {},
				},
			},
		},
	}

	testDynakubeWithSelector = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeSimpleName, Namespace: operatorNamespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{
						NodeSelector: testSelectorLabels,
					},
				},
			},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1beta1.OneAgentStatus{
				Instances: map[string]dynatracev1beta1.OneAgentInstance{
					testNodeWithSelectorName: {},
				},
			},
		},
	}

	caConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: testClusterCa, Namespace: operatorNamespace},
		Data: map[string]string{
			clusterCaKey: testCAValue,
		},
	}

	testSecretDynakubeComplex = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testTokensName, Namespace: operatorNamespace},
		Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
	}

	testSecretDynakubeComplexOnlyApi = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testTokensName, Namespace: operatorNamespace},
		Data:       map[string][]byte{"apiToken": []byte("42")},
	}

	testTlsSecretDynakubeComplex = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "testing", Namespace: operatorNamespace},
		Data:       map[string][]byte{activeGateCaKey: []byte("testing")},
	}

	testSecretDynakubeSimple = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeSimpleName, Namespace: operatorNamespace},
		Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
	}

	kubeNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: kubesystemNamespace, UID: kubesystemUID},
	}

	testNode1 = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: testNode1Name},
	}

	testNode2 = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: testNode2Name},
	}

	testNodeWithLabels = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNodeWithSelectorName,
			Labels: testSelectorLabels,
		},
	}
)

func TestGenerateForNamespace(t *testing.T) {
	t.Run("Add secret for namespace (dynakube with all the fields)", func(t *testing.T) {
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeComplex.Name},
			},
		}
		clt := fake.NewClient(testDynakubeComplex, &testNamespace, testSecretDynakubeComplex, kubeNamespace, caConfigMap, testTlsSecretDynakubeComplex, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)

		_, err := ig.GenerateForNamespace(context.TODO(), *testDynakubeComplex, testNamespace.Name)
		assert.NoError(t, err)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		sercetConfig, ok := initSecret.Data[standalone.SecretConfigFieldName]
		assert.True(t, ok)
		assert.NotNil(t, sercetConfig)
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeSimple.Name},
			},
		}
		clt := fake.NewClient(testDynakubeSimple, &testNamespace, testSecretDynakubeSimple, kubeNamespace, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)

		_, err := ig.GenerateForNamespace(context.TODO(), *testDynakubeSimple, testNamespace.Name)
		assert.NoError(t, err)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		sercetConfig, ok := initSecret.Data[standalone.SecretConfigFieldName]
		assert.True(t, ok)
		assert.NotNil(t, sercetConfig)
	})
}

func TestGenerateForDynakube(t *testing.T) {
	t.Run("Add secret for namespace (dynakube with all the fields)", func(t *testing.T) {
		dk := testDynakubeComplex.DeepCopy()
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeComplex.Name},
			},
		}
		clt := fake.NewClient(&testNamespace, testSecretDynakubeComplex, kubeNamespace, caConfigMap, testTlsSecretDynakubeComplex, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)

		updated, err := ig.GenerateForDynakube(context.TODO(), dk)
		assert.NoError(t, err)
		assert.True(t, updated)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		sercetConfig, ok := initSecret.Data[standalone.SecretConfigFieldName]
		assert.True(t, ok)
		assert.NotNil(t, sercetConfig)
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		dk := testDynakubeSimple.DeepCopy()
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeSimple.Name},
			},
		}
		clt := fake.NewClient(&testNamespace, testSecretDynakubeSimple, kubeNamespace, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)

		updated, err := ig.GenerateForDynakube(context.TODO(), dk)
		assert.NoError(t, err)
		assert.True(t, updated)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		sercetConfig, ok := initSecret.Data[standalone.SecretConfigFieldName]
		assert.True(t, ok)
		assert.NotNil(t, sercetConfig)
	})
	t.Run("Add secret to multiple namespaces (simple dynakube)", func(t *testing.T) {
		dk := testDynakubeSimple.DeepCopy()
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeSimple.Name},
			},
		}
		testOtherNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testOtherNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeSimple.Name},
			},
		}
		clt := fake.NewClient(&testNamespace, &testOtherNamespace, testSecretDynakubeSimple, kubeNamespace, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)

		updated, err := ig.GenerateForDynakube(context.TODO(), dk)
		assert.NoError(t, err)
		assert.True(t, updated)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		sercetConfig, ok := initSecret.Data[standalone.SecretConfigFieldName]
		assert.True(t, ok)
		assert.NotNil(t, sercetConfig)
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testOtherNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		sercetConfig, ok = initSecret.Data[standalone.SecretConfigFieldName]
		assert.True(t, ok)
		assert.NotNil(t, sercetConfig)
	})
}

func TestGetInfraMonitoringNodes(t *testing.T) {
	t.Run("Get IMNodes using nodes", func(t *testing.T) {
		clt := fake.NewClient(testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)
		ig.canWatchNodes = true
		imNodes, err := ig.getHostMonitoringNodes(testDynakubeSimple)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(imNodes))
		assert.Equal(t, testTenantUUID, imNodes[testNode1Name])
		assert.Equal(t, testTenantUUID, imNodes[testNode2Name])
	})
	t.Run("Get IMNodes from dynakubes (without node access)", func(t *testing.T) {
		clt := fake.NewClient()
		ig := NewInitGenerator(clt, clt, operatorNamespace)
		ig.canWatchNodes = false
		imNodes, err := ig.getHostMonitoringNodes(testDynakubeSimple)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(imNodes))
		assert.Equal(t, testTenantUUID, imNodes[testNode2Name])
	})
	t.Run("Get IMNodes from dynakubes with nodeSelector", func(t *testing.T) {
		clt := fake.NewClient(testNodeWithLabels, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace)
		ig.canWatchNodes = true
		imNodes, err := ig.getHostMonitoringNodes(testDynakubeWithSelector)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(imNodes))
		assert.Equal(t, standalone.NoHostTenant, imNodes[testNode1Name])
		assert.Equal(t, standalone.NoHostTenant, imNodes[testNode2Name])
	})
}

func TestPrepareSecretConfigForDynaKube(t *testing.T) {
	t.Run("Create SecretConfig with correct content", func(t *testing.T) {
		testForCorrectContent(t, testSecretDynakubeComplex)
	})
	t.Run("Create SecretConfig with correct content, if only apiToken is provided", func(t *testing.T) {
		testForCorrectContent(t, testSecretDynakubeComplexOnlyApi)
	})
}

func testForCorrectContent(t *testing.T, secret *corev1.Secret) {
	dk := testDynakubeComplex.DeepCopy()
	testNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespaceName,
			Labels: map[string]string{mapper.InstanceLabel: testDynakubeComplex.Name},
		},
	}
	clt := fake.NewClient(&testNamespace, secret, caConfigMap, testTlsSecretDynakubeComplex)
	ig := NewInitGenerator(clt, clt, operatorNamespace)
	imNodes := map[string]string{testNode1Name: testTenantUUID, testNode2Name: testTenantUUID}
	secretConfig, err := ig.prepareSecretConfigForDynaKube(dk, kubesystemUID, imNodes)
	assert.NoError(t, err)
	expectedConfig := standalone.SecretConfig{
		ApiUrl:          dk.Spec.APIURL,
		ApiToken:        string(secret.Data["apiToken"]),
		SkipCertCheck:   dk.Spec.SkipCertCheck,
		Proxy:           testProxy,
		Ca:              testCAValue,
		ClusterID:       string(kubesystemUID),
		TenantUUID:      dk.Status.ConnectionInfo.TenantUUID,
		MonitoringNodes: imNodes,
		HasHost:         true,
		TlsCert:         "testing",
	}
	if content, ok := secret.Data["paasToken"]; ok {
		expectedConfig.PaasToken = string(content)
	} else {
		expectedConfig.PaasToken = expectedConfig.ApiToken
	}
	assert.Equal(t, &expectedConfig, secretConfig)
}
