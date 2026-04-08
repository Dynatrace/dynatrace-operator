package bootstrapperconfig

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPreparePMC(t *testing.T) {
	const (
		testDynakube              = "dk"
		testNamespace             = "ns"
		testProxyURL              = "http://proxy.example.com:8080"
		testHostGroup             = "test-host-group"
		testUUID                  = "uuid"
		testCommunicationEndpoint = "https://mytenant1.dev.dynatracelabs.com:443,https://myag.dev.dynatracelabs.com:443"
		testTenantToken           = "tenant-token"
	)

	t.Run("successfully prepares PMC from API", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					HostGroup:            testHostGroup,
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: testUUID,
						Endpoints:  testCommunicationEndpoint,
					},
				},
			},
		}

		clt := fake.NewClient(
			dk,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := oneagentclientmock.NewAPIClient(t)
		mockDTClient.EXPECT().GetProcessModuleConfig(t.Context()).
			Return(&oneagentclient.ProcessModuleConfig{Properties: []oneagentclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)

		var pmConfig oneagentclient.ProcessModuleConfig
		err = json.Unmarshal(result, &pmConfig)
		require.NoError(t, err)
		assert.Len(t, pmConfig.Properties, 5) // tenantToken, tenantUUID, endpoints, test-property, host-group
	})

	t.Run("successfully prepares PMC with proxy", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					HostGroup:            testHostGroup,
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				Proxy: &value.Source{
					Value: testProxyURL,
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: testUUID,
						Endpoints:  testCommunicationEndpoint,
					},
				},
			},
		}

		clt := fake.NewClient(
			dk,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := oneagentclientmock.NewAPIClient(t)
		mockDTClient.EXPECT().GetProcessModuleConfig(t.Context()).
			Return(&oneagentclient.ProcessModuleConfig{Properties: []oneagentclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)

		var pmConfig oneagentclient.ProcessModuleConfig
		err = json.Unmarshal(result, &pmConfig)
		require.NoError(t, err)

		assert.Len(t, pmConfig.Properties, 6) // tenantToken, tenantUUID, endpoints, test-property, host-group, proxy
	})

	t.Run("error getting PMC from API", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		}

		clt := fake.NewClient(dk)

		mockDTClient := oneagentclientmock.NewAPIClient(t)
		expectedError := errors.New("API error")
		mockDTClient.EXPECT().GetProcessModuleConfig(t.Context()).
			Return(nil, expectedError)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(t.Context(), dk)

		require.Error(t, err)
		require.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})

	t.Run("error getting tenant token", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		}

		clt := fake.NewClient(dk) // No tenant secret

		mockDTClient := oneagentclientmock.NewAPIClient(t)
		mockDTClient.EXPECT().GetProcessModuleConfig(t.Context()).
			Return(&oneagentclient.ProcessModuleConfig{Properties: []oneagentclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(t.Context(), dk)

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("error getting proxy config", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				Proxy: &value.Source{
					ValueFrom: "non-existent-secret", // This will cause an error
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: testUUID,
						Endpoints:  testCommunicationEndpoint,
					},
				},
			},
		}

		clt := fake.NewClient(
			dk,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := oneagentclientmock.NewAPIClient(t)
		mockDTClient.EXPECT().GetProcessModuleConfig(t.Context()).
			Return(&oneagentclient.ProcessModuleConfig{Properties: []oneagentclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(t.Context(), dk)

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("uses cached PMC when not outdated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIurl,
				OneAgent: oneagent.Spec{
					HostGroup:            "test-host-group",
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: testUUID,
						Endpoints:  testCommunicationEndpoint,
					},
				},
			},
		}

		// Set condition as NOT outdated
		k8sconditions.SetSecretCreated(dk.Conditions(), ConfigConditionType, "secret created")

		cachedPMCData, _ := json.Marshal(&oneagentclient.ProcessModuleConfig{Properties: []oneagentclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}})

		sourceSecret := clientSecret(GetSourceConfigSecretName(dk.Name), testNamespace, map[string][]byte{
			pmc.InputFileName: cachedPMCData,
		})

		targetSecret := clientSecret(consts.BootstrapperInitSecretName, testNamespace, map[string][]byte{
			pmc.InputFileName: cachedPMCData,
		})

		clt := fake.NewClient(
			dk,
			sourceSecret,
			targetSecret,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := oneagentclientmock.NewAPIClient(t)
		// Should NOT call GetProcessModuleConfig when using cached data

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(t.Context(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)

		var pmConfig oneagentclient.ProcessModuleConfig
		err = json.Unmarshal(result, &pmConfig)
		require.NoError(t, err)
	})
}
