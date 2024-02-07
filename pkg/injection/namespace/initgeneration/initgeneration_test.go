package initgeneration

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubesystemNamespace = "kube-system"
	kubesystemUID       = types.UID("42")
)

func TestGenerateForNamespace(t *testing.T) {
	t.Run("Add secret for namespace (dynakube with all relevant fields)", func(t *testing.T) {
		dynakube := createDynakube()

		// setup tokens
		apiToken := "api-test"
		paasToken := "paas-test"
		apiTokenSecret := createApiTokenSecret(dynakube, apiToken, paasToken)

		// add TrustedCA
		setTrustedCA(dynakube, "ca-configmap")

		caValue := "ca-test"
		caConfigMap := createTestCaConfigMap(dynakube, caValue)

		// add Proxy
		proxyValue := "proxy-test-value"
		setProxy(dynakube, proxyValue)

		// add TLS secret
		setTlsSecret(dynakube, "tls-test")

		tlsValue := "tls-test-value"
		tlsSecret := createTestTlsSecret(dynakube, tlsValue)

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClient(dynakube, testNamespace, apiTokenSecret, getKubeNamespace(), caConfigMap, tlsSecret)
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		err := ig.GenerateForNamespace(context.TODO(), *dynakube, testNamespace.Name)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, proxyValue)
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		dynakube := createDynakube()

		// setup tokens
		apiToken := "api-test"
		apiTokenSecret := createApiTokenSecret(dynakube, apiToken, apiToken)

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClient(dynakube, testNamespace, apiTokenSecret, getKubeNamespace())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		err := ig.GenerateForNamespace(context.TODO(), *dynakube, testNamespace.Name)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, "")
	})
}

func TestGenerateForDynakube(t *testing.T) {
	t.Run("Add secret for namespace (dynakube with all the fields)", func(t *testing.T) {
		dynakube := createDynakube()

		// setup tokens
		apiToken := "api-test"
		paasToken := "paas-test"
		apiTokenSecret := createApiTokenSecret(dynakube, apiToken, paasToken)

		// add TrustedCA
		setTrustedCA(dynakube, "ca-configmap")

		caValue := "ca-test"
		caConfigMap := createTestCaConfigMap(dynakube, caValue)

		// add Proxy
		proxyValue := "proxy-test-value"
		setProxy(dynakube, proxyValue)

		// add TLS secret
		setTlsSecret(dynakube, "tls-test")

		tlsValue := "tls-test-value"
		tlsSecret := createTestTlsSecret(dynakube, tlsValue)

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret, getKubeNamespace(), caConfigMap, tlsSecret)
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		err := ig.GenerateForDynakube(context.TODO(), dynakube)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, proxyValue)
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		dynakube := createDynakube()
		// setup tokens
		apiToken := "api-test"
		apiTokenSecret := createApiTokenSecret(dynakube, apiToken, apiToken)

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret, getKubeNamespace())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		err := ig.GenerateForDynakube(context.TODO(), dynakube)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, "")
	})
	t.Run("Add secret to multiple namespaces (simple dynakube)", func(t *testing.T) {
		dynakube := createDynakube()
		// setup tokens
		apiToken := "api-test"
		apiTokenSecret := createApiTokenSecret(dynakube, apiToken, apiToken)

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		testOtherNamespace := createTestInjectedNamespace(dynakube, "test-other")
		clt := fake.NewClientWithIndex(testNamespace, testOtherNamespace, apiTokenSecret, getKubeNamespace())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		err := ig.GenerateForDynakube(context.TODO(), dynakube)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, "")

		initSecret = retrieveInitSecret(t, clt, testOtherNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, "")
	})
}

func TestGetInfraMonitoringNodes(t *testing.T) {
	t.Run("Get Monitoring Nodes using nodes", func(t *testing.T) {
		node1 := createTestNode("node-1", nil)
		node2 := createTestNode("node-2", nil)
		dynakube := createDynakube()
		tenantUUID := dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID

		clt := fake.NewClient(node1, node2)
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)
		ig.canWatchNodes = true
		monitoringNodes, err := ig.getHostMonitoringNodes(dynakube)
		require.NoError(t, err)
		assert.Len(t, monitoringNodes, 2)
		assert.Equal(t, tenantUUID, monitoringNodes[node1.Name])
		assert.Equal(t, tenantUUID, monitoringNodes[node2.Name])
	})
	t.Run("Get Monitoring Nodes from dynakubes (without node access)", func(t *testing.T) {
		node1 := createTestNode("node-1", nil)
		node2 := createTestNode("node-2", nil)
		dynakube := createDynakube()
		setNodesToInstances(dynakube, node1.Name, node2.Name)
		tenantUUID := dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID

		clt := fake.NewClient()
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)
		ig.canWatchNodes = false
		monitoringNodes, err := ig.getHostMonitoringNodes(dynakube)
		require.NoError(t, err)
		assert.Len(t, monitoringNodes, 2)
		assert.Equal(t, tenantUUID, monitoringNodes[node1.Name])
		assert.Equal(t, tenantUUID, monitoringNodes[node2.Name])
	})
	t.Run("Get Monitoring Nodes from dynakubes with nodeSelector", func(t *testing.T) {
		node1 := createTestNode("node-1", nil)
		node2 := createTestNode("node-2", nil)
		labeledNode := createTestNode("node-labeled", getTestSelectorLabels())
		dynakube := createDynakube()
		setNodesSelector(dynakube, getTestSelectorLabels())
		tenantUUID := dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID

		clt := fake.NewClient(labeledNode, node1, node2)
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)
		ig.canWatchNodes = true
		monitoringNodes, err := ig.getHostMonitoringNodes(dynakube)
		require.NoError(t, err)
		assert.Len(t, monitoringNodes, 3)
		assert.Equal(t, tenantUUID, monitoringNodes[labeledNode.Name])
		assert.Equal(t, consts.AgentNoHostTenant, monitoringNodes[node1.Name])
		assert.Equal(t, consts.AgentNoHostTenant, monitoringNodes[node2.Name])
	})
}

func TestCreateSecretConfigForDynaKube(t *testing.T) {
	baseDynakube := createDynakube()

	apiToken := "api-test"
	paasToken := "paas-test"
	apiTokenSecret := createApiTokenSecret(baseDynakube, apiToken, paasToken)

	baseExpectedSecretConfig := &startup.SecretConfig{
		ApiUrl:              baseDynakube.ApiUrl(),
		ApiToken:            apiToken,
		PaasToken:           paasToken,
		TenantUUID:          baseDynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID,
		Proxy:               "",
		NoProxy:             "",
		NetworkZone:         "",
		TrustedCAs:          "",
		SkipCertCheck:       false,
		HasHost:             true,
		MonitoringNodes:     nil,
		TlsCert:             "",
		HostGroup:           "",
		InitialConnectRetry: -1,
		CSIMode:             true,
		EnforcementMode:     true,
	}

	t.Run("Create SecretConfig with default content", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with trustedCA", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig

		setTrustedCA(dynakube, "ca-configmap")

		caValue := "ca-test"
		caConfigMap := createTestCaConfigMap(dynakube, caValue)
		expectedSecretConfig.TrustedCAs = caValue

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy(), caConfigMap.DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with proxy", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		proxyValue := "proxy-test-value"
		setProxy(dynakube, proxyValue)
		expectedSecretConfig.Proxy = proxyValue

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig without proxy if feature-flag is set", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		proxyValue := "proxy-test-value"
		setProxy(dynakube, proxyValue)
		setAnnotation(dynakube, map[string]string{
			dynatracev1beta1.AnnotationFeatureOneAgentIgnoreProxy: "true",
		})

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with no-proxy", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		proxyValue := "proxy-test-value"
		setNoProxy(dynakube, proxyValue)
		expectedSecretConfig.NoProxy = proxyValue

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with initial connect retry", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		retryValue := "123"
		setInitialConnectRetry(dynakube, retryValue)

		expectedSecretConfig.InitialConnectRetry = 123

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with tlsSecret", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		setTlsSecret(dynakube, "tls-test")

		expectedSecretConfig := *baseExpectedSecretConfig
		tlsValue := "tls-test-value"
		tlsSecret := createTestTlsSecret(dynakube, tlsValue)
		expectedSecretConfig.TlsCert = tlsValue
		// since we have ActiveGate we add it by default as noProxy
		expectedSecretConfig.OneAgentNoProxy = "dynakube-test-activegate.dynatrace-test"

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy(), tlsSecret)
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with networkZone", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		networkZone := "test-network"
		dynakube.Spec.NetworkZone = networkZone
		expectedSecretConfig.NetworkZone = networkZone

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with skipCertCheck", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		dynakube.Spec.SkipCertCheck = true
		expectedSecretConfig.SkipCertCheck = true

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with monitoring node", func(t *testing.T) {
		dynakube := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		monitoringNodes := map[string]string{
			"node-1": "tenant-1",
		}
		expectedSecretConfig.MonitoringNodes = monitoringNodes

		testNamespace := createTestInjectedNamespace(dynakube, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dynakube.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dynakube, monitoringNodes)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})
}

func getTestSelectorLabels() map[string]string {
	return map[string]string{"test": "label"}
}

func createDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "dynakube-test",
			Namespace:   "dynatrace-test",
			Annotations: map[string]string{},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://test-url/e/tenant/api",
			Tokens: "dynakube-test",
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
				}},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				ConnectionInfoStatus: dynatracev1beta1.OneAgentConnectionInfoStatus{
					ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
						TenantUUID:  "test-tenant",
						Endpoints:   "beep.com;bop.com",
						LastRequest: metav1.Time{},
					},
				},
			},
		},
	}
}

func setProxy(dynakube *dynatracev1beta1.DynaKube, value string) {
	dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: value}
}

func setAnnotation(dynakube *dynatracev1beta1.DynaKube, value map[string]string) {
	dynakube.ObjectMeta.Annotations = value
}

func setTrustedCA(dynakube *dynatracev1beta1.DynaKube, value string) {
	dynakube.Spec.TrustedCAs = value
}

func checkProxy(t *testing.T, generatedSecret corev1.Secret, expectedValue string) {
	proxy, ok := generatedSecret.Data[dynatracev1beta1.ProxyKey]
	require.True(t, ok)
	assert.NotNil(t, proxy)
	assert.Equal(t, expectedValue, string(proxy))
}

func setNoProxy(dynakube *dynatracev1beta1.DynaKube, value string) {
	dynakube.Annotations[dynatracev1beta1.AnnotationFeatureNoProxy] = value
}

func setInitialConnectRetry(dynakube *dynatracev1beta1.DynaKube, value string) {
	dynakube.Annotations[dynatracev1beta1.AnnotationFeatureOneAgentInitialConnectRetry] = value
}

func setTlsSecret(dynakube *dynatracev1beta1.DynaKube, value string) {
	dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
		Capabilities: []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.KubeMonCapability.DisplayName,
		},
		TlsSecretName: value,
	}
}

func setNodesToInstances(dynakube *dynatracev1beta1.DynaKube, nodeNames ...string) {
	instances := map[string]dynatracev1beta1.OneAgentInstance{}
	for _, name := range nodeNames {
		instances[name] = dynatracev1beta1.OneAgentInstance{}
	}

	dynakube.Status.OneAgent.Instances = instances
}

func setNodesSelector(dynakube *dynatracev1beta1.DynaKube, selector map[string]string) {
	dynakube.Spec.OneAgent.CloudNativeFullStack.NodeSelector = selector
}

func createTestCaConfigMap(dynakube *dynatracev1beta1.DynaKube, value string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: dynakube.Spec.TrustedCAs, Namespace: dynakube.Namespace},
		Data: map[string]string{
			dynatracev1beta1.TrustedCAKey: value,
		},
	}
}

func createTestTlsSecret(dynakube *dynatracev1beta1.DynaKube, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dynakube.Spec.ActiveGate.TlsSecretName, Namespace: dynakube.Namespace},
		Data:       map[string][]byte{dynatracev1beta1.TlsCertKey: []byte(value)},
	}
}

func getKubeNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: kubesystemNamespace, UID: kubesystemUID},
	}
}

func createApiTokenSecret(dynakube *dynatracev1beta1.DynaKube, apiToken, paasToken string) *corev1.Secret {
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dynakube.Name, Namespace: dynakube.Namespace},
		Data:       map[string][]byte{},
	}
	if apiToken != "" {
		tokenSecret.Data["apiToken"] = []byte(apiToken)
	}

	if paasToken != "" {
		tokenSecret.Data["paasToken"] = []byte(paasToken)
	}

	return tokenSecret
}

func createTestNode(name string, selector map[string]string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: selector,
		},
	}
}

func createTestInjectedNamespace(dynakube *dynatracev1beta1.DynaKube, name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{dtwebhook.InjectionInstanceLabel: dynakube.Name},
		},
	}
}

func retrieveInitSecret(t *testing.T, clt client.Client, namespaceName string) corev1.Secret {
	var initSecret corev1.Secret
	err := clt.Get(context.TODO(), types.NamespacedName{Name: consts.AgentInitSecretName, Namespace: namespaceName}, &initSecret)
	require.NoError(t, err)
	assert.Len(t, initSecret.Data, 2)

	return initSecret
}

func checkSecretConfigExists(t *testing.T, initSecret corev1.Secret) {
	secretConfig, ok := initSecret.Data[consts.AgentInitSecretConfigField]
	require.True(t, ok)
	require.NotNil(t, secretConfig)

	var parsedConfig startup.SecretConfig
	err := json.Unmarshal(secretConfig, &parsedConfig)
	require.NoError(t, err)
}
