package initgeneration

import (
	"context"
	_ "embed"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//go:embed init.sh.test-sample
var scriptSample string

var (
	operatorNamespace        = "dynatrace"
	testNamespaceName        = "namespace"
	testOtherNamespaceName   = "other-namespace"
	testDynakubeComplexName  = "dynakubeComplex"
	testDynakubeSimpleName   = "dynakubeSimple"
	testTokensName           = "kitchen-sink"
	testApiUrl               = "https://test-url/api"
	testProxy                = "testproxy.com"
	testtrustCAsCM           = "testtrustedCAsConfigMap"
	testCAValue              = "somecertificate"
	testTenantUUID           = "abc12345"
	kubesystemNamespace      = "kube-system"
	kubesystemUID            = types.UID("42")
	testNode1Name            = "node1"
	testNode2Name            = "node2"
	testNodeWithSelectorName = "nodeWselector"
	testSelectorLabels       = map[string]string{"test": "label"}

	testDynakubeComplex = &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeComplexName, Namespace: operatorNamespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:     testApiUrl,
			Proxy:      &dynatracev1alpha1.DynaKubeProxy{Value: testProxy},
			TrustedCAs: testtrustCAsCM,
			Tokens:     testTokensName,
			InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
				FullStackSpec: dynatracev1alpha1.FullStackSpec{Enabled: true},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{
					testNode1Name: {},
				},
			},
		},
	}

	testDynakubeSimple = &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeSimpleName, Namespace: operatorNamespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: testApiUrl,
			InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
				FullStackSpec: dynatracev1alpha1.FullStackSpec{Enabled: true},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{
					testNode2Name: {},
				},
			},
		},
	}

	testDynakubeWithSelector = &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynakubeSimpleName, Namespace: operatorNamespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: testApiUrl,
			InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
				FullStackSpec: dynatracev1alpha1.FullStackSpec{Enabled: true, NodeSelector: testSelectorLabels},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			ConnectionInfo: dynatracev1alpha1.ConnectionInfoStatus{
				TenantUUID: testTenantUUID,
			},
		},
	}

	caConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: testtrustCAsCM, Namespace: operatorNamespace},
		Data: map[string]string{
			"certs": testCAValue,
		},
	}

	testSecretDynakubeComplex = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testTokensName, Namespace: operatorNamespace},
		Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
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
		clt := fake.NewClient(testDynakubeComplex, &testNamespace, testSecretDynakubeComplex, kubeNamespace, caConfigMap, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())

		err := ig.GenerateForNamespace(context.TODO(), testDynakubeComplex.Name, testNamespace.Name)
		assert.NoError(t, err)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(initSecret.Data))
		initSh, ok := initSecret.Data["init.sh"]
		assert.True(t, ok)
		assert.NotNil(t, initSh)
		proxy, ok := initSecret.Data["proxy"]
		assert.True(t, ok)
		assert.Equal(t, testProxy, string(proxy))
		ca, ok := initSecret.Data["ca.pem"]
		assert.True(t, ok)
		assert.Equal(t, testCAValue, string(ca))
	})
	t.Run("Add secret for namespace (simple dynakube)", func(t *testing.T) {
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeSimple.Name},
			},
		}
		clt := fake.NewClient(testDynakubeSimple, &testNamespace, testSecretDynakubeSimple, kubeNamespace, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())

		err := ig.GenerateForNamespace(context.TODO(), testDynakubeSimple.Name, testNamespace.Name)
		assert.NoError(t, err)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		initSh, ok := initSecret.Data["init.sh"]
		assert.True(t, ok)
		assert.NotNil(t, initSh)
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
		clt := fake.NewClient(&testNamespace, testSecretDynakubeComplex, kubeNamespace, caConfigMap, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())

		updated, err := ig.GenerateForDynakube(context.TODO(), dk)
		assert.NoError(t, err)
		assert.True(t, updated)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(initSecret.Data))
		initSh, ok := initSecret.Data["init.sh"]
		assert.True(t, ok)
		assert.NotNil(t, initSh)
		proxy, ok := initSecret.Data["proxy"]
		assert.True(t, ok)
		assert.Equal(t, testProxy, string(proxy))
		ca, ok := initSecret.Data["ca.pem"]
		assert.True(t, ok)
		assert.Equal(t, testCAValue, string(ca))
		assert.NotNil(t, dk.Status.LastInitSecretHash)
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
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())

		updated, err := ig.GenerateForDynakube(context.TODO(), dk)
		assert.NoError(t, err)
		assert.True(t, updated)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		initSh, ok := initSecret.Data["init.sh"]
		assert.True(t, ok)
		assert.NotNil(t, initSh)
		assert.NotNil(t, dk.Status.LastInitSecretHash)
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
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())

		updated, err := ig.GenerateForDynakube(context.TODO(), dk)
		assert.NoError(t, err)
		assert.True(t, updated)

		var initSecret corev1.Secret
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		initSh, ok := initSecret.Data["init.sh"]
		assert.True(t, ok)
		assert.NotNil(t, initSh)
		err = clt.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretConfigName, Namespace: testOtherNamespace.Name}, &initSecret)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(initSecret.Data))
		initSh, ok = initSecret.Data["init.sh"]
		assert.True(t, ok)
		assert.NotNil(t, initSh)
		assert.NotNil(t, dk.Status.LastInitSecretHash)
	})
}

func TestGetInfraMonitoringNodes(t *testing.T) {
	t.Run("Get IMNodes from multiple dynakubes", func(t *testing.T) {
		clt := fake.NewClient(testDynakubeComplex, testDynakubeSimple, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())
		imNodes, err := ig.getInfraMonitoringNodes(testDynakubeSimple)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(imNodes))
		assert.Equal(t, testTenantUUID, imNodes[testNode1Name])
		assert.Equal(t, testTenantUUID, imNodes[testNode2Name])
	})
	t.Run("Get IMNodes from dynakubes with nodeSelector", func(t *testing.T) {
		clt := fake.NewClient(testNodeWithLabels, testDynakubeWithSelector, testNode1, testNode2)
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())
		imNodes, err := ig.getInfraMonitoringNodes(testDynakubeWithSelector)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(imNodes))
		assert.Equal(t, notMappedIM, imNodes[testNode1Name])
		assert.Equal(t, notMappedIM, imNodes[testNode2Name])
	})
}

func TestPrepareScriptForDynaKube(t *testing.T) {
	t.Run("Create init.sh with correct content", func(t *testing.T) {
		dk := testDynakubeComplex.DeepCopy()
		testNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespaceName,
				Labels: map[string]string{mapper.InstanceLabel: testDynakubeComplex.Name},
			},
		}
		clt := fake.NewClient(&testNamespace, testSecretDynakubeComplex, caConfigMap)
		ig := NewInitGenerator(clt, clt, operatorNamespace, logger.NewDTLogger())
		imNodes := map[string]string{testNode1Name: testTenantUUID, testNode2Name: testTenantUUID}
		sc, err := ig.prepareScriptForDynaKube(dk, kubesystemUID, imNodes)
		assert.NoError(t, err)
		expectedScript := script{
			ApiUrl:        dk.Spec.APIURL,
			SkipCertCheck: dk.Spec.SkipCertCheck,
			PaaSToken:     "42",
			Proxy:         testProxy,
			TrustedCAs:    []byte(testCAValue),
			ClusterID:     string(kubesystemUID),
			TenantUUID:    dk.Status.ConnectionInfo.TenantUUID,
			IMNodes:       imNodes,
		}
		assert.Equal(t, &expectedScript, sc)

		initSh, err := sc.generate()
		assert.NoError(t, err)
		assert.Equal(t, scriptSample, string(initSh["init.sh"]))
	})
}
