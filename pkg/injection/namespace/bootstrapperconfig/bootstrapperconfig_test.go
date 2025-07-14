package bootstrapperconfig

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/enrichment/endpoint"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/curl"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken       = "test-paas-token"
	testAPIToken        = "test-api-token"
	testDataIngestToken = "test-ingest-token"

	testUUID                  = "test-uuid"
	testTenantToken           = "abcd"
	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"

	testHost = "test-host"

	testDynakube   = "test-dynakube"
	testNamespace  = "test-namespace"
	testNamespace2 = "test-namespace2"

	testNamespaceDynatrace = "dynatrace"

	testAPIurl = "https://" + testHost + "/e/" + testUUID + "/api"

	oldCertValue = "old-cert-value"
	oldTrustedCa = "old-trusted-ca"
)

func TestNewSecretGenerator(t *testing.T) {
	client := fake.NewClient()
	mockDTClient := dtclientmock.NewClient(t)

	secretGenerator := NewSecretGenerator(client, client, mockDTClient)
	assert.NotNil(t, secretGenerator)

	assert.Equal(t, client, secretGenerator.client)
	assert.Equal(t, client, secretGenerator.apiReader)
	assert.Equal(t, mockDTClient, secretGenerator.dtClient)
}

func TestGenerateForDynakube(t *testing.T) {
	t.Run("succcessfully generate config secret for dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.APIToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespaceDynatrace, map[string][]byte{
				"tenant-token": []byte(testTenantToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(context.Background(), dk)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secret)
		require.NoError(t, err)
		require.Equal(t, consts.BootstrapperInitSecretName, secret.Name)
		assert.NotEmpty(t, secret.Data)

		var sourceSecret corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}, &sourceSecret)
		require.NoError(t, err)

		require.Equal(t, GetSourceConfigSecretName(dk.Name), sourceSecret.Name)
		assert.Equal(t, secret.Data, sourceSecret.Data)

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionTrue, c.Status)
	})
	t.Run("successfully generate secret with fields for dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
				Annotations: map[string]string{
					exp.OAInitialConnectRetryKey: "6500",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:     testAPIurl,
				TrustedCAs: "test-trusted-ca",
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(true),
				},
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					TLSSecretName: "test-tls-secret-name",
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.APIToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
			clientSecret(dk.ActiveGate().TLSSecretName, testNamespaceDynatrace, map[string][]byte{
				dynakube.TLSCertKey: []byte("test-cert-value"),
			}),
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespaceDynatrace, map[string][]byte{
				"tenant-token": []byte(testTenantToken),
			}),
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-trusted-ca",
					Namespace: testNamespaceDynatrace,
				},
				Data: map[string]string{
					dynakube.TrustedCAKey: "test-trusted-ca-value",
				},
			},
		)

		mockDTClient := dtclientmock.NewClient(t)

		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(context.Background(), dk)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secret)
		require.NoError(t, err)
		require.NotEmpty(t, secret)
		assert.Equal(t, consts.BootstrapperInitSecretName, secret.Name)

		_, ok := secret.Data[pmc.InputFileName]
		require.True(t, ok)

		_, ok = secret.Data[curl.InputFileName]
		require.True(t, ok)

		_, ok = secret.Data[endpoint.InputFileName]
		require.True(t, ok)

		// check certs secret
		var secretCerts corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitCertsSecretName, Namespace: testNamespace}, &secretCerts)
		require.NoError(t, err)
		require.NotEmpty(t, secretCerts)
		assert.Equal(t, consts.BootstrapperInitCertsSecretName, secretCerts.Name)

		_, ok = secretCerts.Data[ca.TrustedCertsInputFile]
		require.True(t, ok)

		_, ok = secretCerts.Data[ca.AgCertsInputFile]
		require.True(t, ok)

		var sourceSecret corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}, &sourceSecret)
		require.NoError(t, err)

		require.Equal(t, GetSourceConfigSecretName(dk.Name), sourceSecret.Name)
		assert.Equal(t, secret.Data, sourceSecret.Data)

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionTrue, c.Status)
	})
	t.Run("update secret with preexisting secret + fields", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
				Annotations: map[string]string{
					exp.OAInitialConnectRetryKey:     "6500",
					exp.AGAutomaticTLSCertificateKey: "false",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(true),
				},
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.APIToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
			clientSecret(dk.ActiveGate().TLSSecretName, testNamespaceDynatrace, map[string][]byte{
				dynakube.TLSCertKey: []byte("test-cert-value"),
			}),
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespaceDynatrace, map[string][]byte{
				"tenant-token": []byte(testTenantToken),
			}),
			clientSecret(consts.BootstrapperInitSecretName, testNamespace, map[string][]byte{
				pmc.InputFileName:        nil,
				ca.TrustedCertsInputFile: []byte(oldTrustedCa),
				ca.AgCertsInputFile:      []byte(oldCertValue),
			}),
			clientSecret(GetSourceConfigSecretName(dk.Name), dk.Namespace, map[string][]byte{
				pmc.InputFileName:        nil,
				ca.TrustedCertsInputFile: []byte(oldTrustedCa),
				ca.AgCertsInputFile:      []byte(oldCertValue),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(context.Background(), dk)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secret)
		require.NoError(t, err)

		require.NotEmpty(t, secret)

		assert.Equal(t, consts.BootstrapperInitSecretName, secret.Name)
		_, ok := secret.Data[pmc.InputFileName]
		require.True(t, ok)

		_, ok = secret.Data[ca.TrustedCertsInputFile]
		require.False(t, ok)

		_, ok = secret.Data[ca.AgCertsInputFile]
		require.False(t, ok)

		_, ok = secret.Data[curl.InputFileName]
		require.True(t, ok)

		_, ok = secret.Data[endpoint.InputFileName]
		require.True(t, ok)

		var sourceSecret corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}, &sourceSecret)
		require.NoError(t, err)

		require.Equal(t, GetSourceConfigSecretName(dk.Name), sourceSecret.Name)
		assert.Equal(t, secret.Data, sourceSecret.Data)

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionTrue, c.Status)
	})
	t.Run("fail while generating secret for dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.APIToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(context.Background(), dk)
		require.Error(t, err)

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)
	})
}

func TestCleanup(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIurl,
		},
		Status: dynakube.DynaKubeStatus{
			Conditions: []metav1.Condition{
				{Type: ConfigConditionType},
				{Type: "other"},
			},
		},
	}

	clt := fake.NewClientWithIndex(
		dk,
		clientInjectedNamespace(testNamespace, testDynakube),
		clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
			dtclient.APIToken:  []byte(testAPIToken),
			dtclient.PaasToken: []byte(testPaasToken),
		}),
		clientSecret(dk.OneAgent().GetTenantSecret(), testNamespaceDynatrace, map[string][]byte{
			"tenant-token": []byte(testTenantToken),
		}),
		clientSecret(consts.BootstrapperInitSecretName, testNamespace, nil),
		clientSecret(consts.BootstrapperInitSecretName, testNamespace2, nil),
		clientSecret(GetSourceConfigSecretName(dk.Name), dk.Namespace, nil),
	)
	namespaces := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
		{ObjectMeta: metav1.ObjectMeta{Name: testNamespace2}},
	}

	var secretNS1 corev1.Secret
	err := clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secretNS1)
	require.NoError(t, err)

	require.NotEmpty(t, secretNS1)
	assert.Equal(t, consts.BootstrapperInitSecretName, secretNS1.Name)

	var secretNS2 corev1.Secret
	err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secretNS2)
	require.NoError(t, err)

	require.NotEmpty(t, secretNS2)
	assert.Equal(t, consts.BootstrapperInitSecretName, secretNS2.Name)

	err = Cleanup(context.Background(), clt, clt, namespaces, dk)
	require.NoError(t, err)

	var deleted corev1.Secret
	err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &deleted)
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace2}, &deleted)
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	err = clt.Get(context.Background(), client.ObjectKey{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}, &deleted)
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	require.Nil(t, meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType))
}

func clientSecret(secretName string, namespaceName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
		},
		Data: data,
	}
}

func clientInjectedNamespace(namespaceName string, dynakubeName string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "corev1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: dynakubeName,
			},
		},
	}
}
