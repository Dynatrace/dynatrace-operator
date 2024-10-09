package initgeneration

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
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
		dk := createDynakube()

		// setup tokens
		apiToken := "api-test"
		paasToken := "paas-test"
		apiTokenSecret := createApiTokenSecret(dk, apiToken, paasToken)

		// add TrustedCA
		setTrustedCA(dk, "ca-configmap")

		caValue := "ca-test"
		caConfigMap := createTestCaConfigMap(dk, caValue)

		// add Proxy
		proxyValue := "proxy-test-value"
		setProxy(dk, proxyValue)

		// add TLS secret
		setTlsSecret(dk, "tls-test")

		tlsValue := "tls-test-value"
		tlsSecret := createTestTlsSecret(dk, tlsValue)

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClient(dk, testNamespace, apiTokenSecret, getKubeNamespace(), caConfigMap, tlsSecret)
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		err := ig.GenerateForNamespace(context.TODO(), *dk, testNamespace.Name)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, proxyValue)
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		dk := createDynakube()

		// setup tokens
		apiToken := "api-test"
		apiTokenSecret := createApiTokenSecret(dk, apiToken, apiToken)

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClient(dk, testNamespace, apiTokenSecret, getKubeNamespace())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		err := ig.GenerateForNamespace(context.TODO(), *dk, testNamespace.Name)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, "")
	})
}

func TestGenerateForDynakube(t *testing.T) {
	t.Run("Add secret for namespace (dynakube with all the fields)", func(t *testing.T) {
		dk := createDynakube()

		// setup tokens
		apiToken := "api-test"
		paasToken := "paas-test"
		apiTokenSecret := createApiTokenSecret(dk, apiToken, paasToken)

		// add TrustedCA
		setTrustedCA(dk, "ca-configmap")

		caValue := "ca-test"
		caConfigMap := createTestCaConfigMap(dk, caValue)

		// add Proxy
		proxyValue := "proxy-test-value"
		setProxy(dk, proxyValue)

		// add TLS secret
		setTlsSecret(dk, "tls-test")

		tlsValue := "tls-test-value"
		tlsSecret := createTestTlsSecret(dk, tlsValue)

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret, getKubeNamespace(), caConfigMap, tlsSecret)
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		err := ig.GenerateForDynakube(context.TODO(), dk)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, proxyValue)
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		dk := createDynakube()
		// setup tokens
		apiToken := "api-test"
		apiTokenSecret := createApiTokenSecret(dk, apiToken, apiToken)

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret, getKubeNamespace())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		err := ig.GenerateForDynakube(context.TODO(), dk)
		require.NoError(t, err)

		initSecret := retrieveInitSecret(t, clt, testNamespace.Name)
		checkSecretConfigExists(t, initSecret)
		checkProxy(t, initSecret, "")
	})
	t.Run("Add secret to multiple namespaces (simple dynakube)", func(t *testing.T) {
		dk := createDynakube()
		// setup tokens
		apiToken := "api-test"
		apiTokenSecret := createApiTokenSecret(dk, apiToken, apiToken)

		testNamespace := createTestInjectedNamespace(dk, "test")
		testOtherNamespace := createTestInjectedNamespace(dk, "test-other")
		clt := fake.NewClientWithIndex(testNamespace, testOtherNamespace, apiTokenSecret, getKubeNamespace())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		err := ig.GenerateForDynakube(context.TODO(), dk)
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
		dk := createDynakube()
		tenantUUID := dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID

		clt := fake.NewClient(node1, node2)
		ig := NewInitGenerator(clt, clt, dk.Namespace)
		ig.canWatchNodes = true
		monitoringNodes, err := ig.getHostMonitoringNodes(dk)
		require.NoError(t, err)
		assert.Len(t, monitoringNodes, 2)
		assert.Equal(t, tenantUUID, monitoringNodes[node1.Name])
		assert.Equal(t, tenantUUID, monitoringNodes[node2.Name])
	})
	t.Run("Get Monitoring Nodes from dynakubes (without node access)", func(t *testing.T) {
		node1 := createTestNode("node-1", nil)
		node2 := createTestNode("node-2", nil)
		dk := createDynakube()
		setNodesToInstances(dk, node1.Name, node2.Name)
		tenantUUID := dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID

		clt := fake.NewClient()
		ig := NewInitGenerator(clt, clt, dk.Namespace)
		ig.canWatchNodes = false
		monitoringNodes, err := ig.getHostMonitoringNodes(dk)
		require.NoError(t, err)
		assert.Len(t, monitoringNodes, 2)
		assert.Equal(t, tenantUUID, monitoringNodes[node1.Name])
		assert.Equal(t, tenantUUID, monitoringNodes[node2.Name])
	})
	t.Run("Get Monitoring Nodes from dynakubes with nodeSelector", func(t *testing.T) {
		node1 := createTestNode("node-1", nil)
		node2 := createTestNode("node-2", nil)
		labeledNode := createTestNode("node-labeled", getTestSelectorLabels())
		dk := createDynakube()
		setNodesSelector(dk, getTestSelectorLabels())
		tenantUUID := dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID

		clt := fake.NewClient(labeledNode, node1, node2)
		ig := NewInitGenerator(clt, clt, dk.Namespace)
		ig.canWatchNodes = true
		monitoringNodes, err := ig.getHostMonitoringNodes(dk)
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
		SkipCertCheck:       false,
		HasHost:             true,
		MonitoringNodes:     nil,
		HostGroup:           "",
		InitialConnectRetry: -1,
		CSIMode:             true,
		EnforcementMode:     true,
	}

	t.Run("Create SecretConfig with default content", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with trustedCA", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig

		setTrustedCA(dk, "ca-configmap")

		caValue := "ca-test"
		caConfigMap := createTestCaConfigMap(dk, caValue)

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy(), caConfigMap.DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretData, err := ig.generate(context.TODO(), dk)
		require.NoError(t, err)

		_, ok := secretData[consts.AgentInitSecretConfigField]
		require.True(t, ok)

		var secretConfig startup.SecretConfig
		err = json.Unmarshal(secretData[consts.AgentInitSecretConfigField], &secretConfig)
		require.NoError(t, err)

		require.Empty(t, secretConfig.MonitoringNodes)
		secretConfig.MonitoringNodes = nil

		assert.Equal(t, expectedSecretConfig, secretConfig)
		assert.Equal(t, caValue, string(secretData[consts.TrustedCAsInitSecretField]))
	})

	t.Run("Create SecretConfig with proxy", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		proxyValue := "proxy-test-value"
		setProxy(dk, proxyValue)
		expectedSecretConfig.Proxy = proxyValue

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig without proxy if feature-flag is set", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		proxyValue := "proxy-test-value"
		setProxy(dk, proxyValue)
		setAnnotation(dk, map[string]string{
			dynakube.AnnotationFeatureOneAgentIgnoreProxy: "true", //nolint:staticcheck
		})

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with no-proxy", func(t *testing.T) {
		proxyValue := "proxy-test-value"
		noProxyValue := "no-proxy-test-value"
		dk := baseDynakube.DeepCopy()
		dk.Spec.Proxy = &value.Source{Value: proxyValue}
		setNoProxy(dk, noProxyValue)

		expectedSecretConfig := *baseExpectedSecretConfig
		expectedSecretConfig.NoProxy = noProxyValue
		expectedSecretConfig.OneAgentNoProxy = noProxyValue
		expectedSecretConfig.Proxy = proxyValue

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with no-proxy + activegate", func(t *testing.T) {
		proxyValue := "proxy-test-value"
		noProxyValue := "no-proxy-test-value"
		dk := baseDynakube.DeepCopy()
		dk.Spec.Proxy = &value.Source{Value: proxyValue}
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
		}
		setNoProxy(dk, noProxyValue)

		expectedSecretConfig := *baseExpectedSecretConfig
		expectedSecretConfig.NoProxy = noProxyValue
		expectedSecretConfig.OneAgentNoProxy = noProxyValue + ",dynakube-test-activegate.dynatrace-test"
		expectedSecretConfig.Proxy = proxyValue

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with initial connect retry", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		retryValue := "123"
		setInitialConnectRetry(dk, retryValue)

		expectedSecretConfig.InitialConnectRetry = 123

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with tlsSecret", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		setTlsSecret(dk, "tls-test")
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}

		expectedSecretConfig := *baseExpectedSecretConfig
		tlsValue := "tls-test-value"
		tlsSecret := createTestTlsSecret(dk, tlsValue)

		// since we have ActiveGate we add it by default as noProxy
		expectedSecretConfig.OneAgentNoProxy = "dynakube-test-activegate.dynatrace-test"

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy(), tlsSecret)
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretData, err := ig.generate(context.TODO(), dk)
		require.NoError(t, err)

		var secretConfig startup.SecretConfig
		err = json.Unmarshal(secretData[consts.AgentInitSecretConfigField], &secretConfig)
		require.NoError(t, err)

		require.Empty(t, secretConfig.MonitoringNodes)
		secretConfig.MonitoringNodes = nil

		assert.Equal(t, expectedSecretConfig, secretConfig)
		assert.Equal(t, tlsValue, string(secretData[consts.ActiveGateCAsInitSecretField]))
	})

	t.Run("Create SecretConfig with networkZone", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		networkZone := "test-network"
		dk.Spec.NetworkZone = networkZone
		expectedSecretConfig.NetworkZone = networkZone

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with skipCertCheck", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		dk.Spec.SkipCertCheck = true
		expectedSecretConfig.SkipCertCheck = true

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, nil)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})

	t.Run("Create SecretConfig with monitoring node", func(t *testing.T) {
		dk := baseDynakube.DeepCopy()
		expectedSecretConfig := *baseExpectedSecretConfig
		monitoringNodes := map[string]string{
			"node-1": "tenant-1",
		}
		expectedSecretConfig.MonitoringNodes = monitoringNodes

		testNamespace := createTestInjectedNamespace(dk, "test")
		clt := fake.NewClientWithIndex(testNamespace, apiTokenSecret.DeepCopy(), getKubeNamespace().DeepCopy())
		ig := NewInitGenerator(clt, clt, dk.Namespace)

		secretConfig, err := ig.createSecretConfigForDynaKube(context.TODO(), dk, monitoringNodes)
		require.NoError(t, err)
		assert.Equal(t, expectedSecretConfig, *secretConfig)
	})
}

func getTestSelectorLabels() map[string]string {
	return map[string]string{"test": "label"}
}

func createDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "dynakube-test",
			Namespace:   "dynatrace-test",
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://test-url/e/tenant/api",
			Tokens: "dynakube-test",
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
					HostInjectSpec: dynakube.HostInjectSpec{},
				}},
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: dynakube.OneAgentStatus{
				ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: "test-tenant",
						Endpoints:  "beep.com;bop.com",
					},
				},
			},
		},
	}
}

func setProxy(dk *dynakube.DynaKube, proxyValue string) {
	dk.Spec.Proxy = &value.Source{Value: proxyValue}
}

func setAnnotation(dk *dynakube.DynaKube, value map[string]string) {
	dk.ObjectMeta.Annotations = value
}

func setTrustedCA(dk *dynakube.DynaKube, value string) {
	dk.Spec.TrustedCAs = value
}

func checkProxy(t *testing.T, generatedSecret corev1.Secret, expectedValue string) {
	proxy, ok := generatedSecret.Data[dynakube.ProxyKey]
	require.True(t, ok)
	assert.NotNil(t, proxy)
	assert.Equal(t, expectedValue, string(proxy))
}

func setNoProxy(dk *dynakube.DynaKube, value string) {
	dk.Annotations[dynakube.AnnotationFeatureNoProxy] = value
}

func setInitialConnectRetry(dk *dynakube.DynaKube, value string) {
	dk.Annotations[dynakube.AnnotationFeatureOneAgentInitialConnectRetry] = value
}

func setTlsSecret(dk *dynakube.DynaKube, value string) {
	dk.Spec.ActiveGate = activegate.Spec{
		Capabilities: []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
		},
		TlsSecretName: value,
	}
}

func setNodesToInstances(dk *dynakube.DynaKube, nodeNames ...string) {
	instances := map[string]dynakube.OneAgentInstance{}
	for _, name := range nodeNames {
		instances[name] = dynakube.OneAgentInstance{}
	}

	dk.Status.OneAgent.Instances = instances
}

func setNodesSelector(dk *dynakube.DynaKube, selector map[string]string) {
	dk.Spec.OneAgent.CloudNativeFullStack.NodeSelector = selector
}

func createTestCaConfigMap(dk *dynakube.DynaKube, value string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: dk.Spec.TrustedCAs, Namespace: dk.Namespace},
		Data: map[string]string{
			dynakube.TrustedCAKey: value,
		},
	}
}

func createTestTlsSecret(dk *dynakube.DynaKube, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dk.Spec.ActiveGate.TlsSecretName, Namespace: dk.Namespace},
		Data:       map[string][]byte{dynakube.TLSCertKey: []byte(value)},
	}
}

func getKubeNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: kubesystemNamespace, UID: kubesystemUID},
	}
}

func createApiTokenSecret(dk *dynakube.DynaKube, apiToken, paasToken string) *corev1.Secret {
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dk.Name, Namespace: dk.Namespace},
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

func createTestInjectedNamespace(dk *dynakube.DynaKube, name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{dtwebhook.InjectionInstanceLabel: dk.Name},
		},
	}
}

func retrieveInitSecret(t *testing.T, clt client.Client, namespaceName string) corev1.Secret {
	var initSecret corev1.Secret
	err := clt.Get(context.TODO(), types.NamespacedName{Name: consts.AgentInitSecretName, Namespace: namespaceName}, &initSecret)
	require.NoError(t, err)
	assert.Len(t, initSecret.Data, 4) // agcerts, config, proxy, trustedcas

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
