package bootstrapperconfig

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testProxyURL = "http://proxy.example.com:8080"
)

func TestPreparePMC(t *testing.T) {
	t.Run("successfully prepares PMC from API", func(t *testing.T) {
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
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: testUUID,
							Endpoints:  testCommunicationEndpoint,
						},
					},
				},
			},
		}

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated")

		clt := fake.NewClient(
			dk,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).
			Return(&dtclient.ProcessModuleConfig{Properties: []dtclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(context.Background(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)

		var pmConfig dtclient.ProcessModuleConfig
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
					HostGroup:            "test-host-group",
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				Proxy: &value.Source{
					Value: testProxyURL,
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: testUUID,
							Endpoints:  testCommunicationEndpoint,
						},
					},
				},
			},
		}

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated")

		clt := fake.NewClient(
			dk,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).
			Return(&dtclient.ProcessModuleConfig{Properties: []dtclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(context.Background(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)

		var pmConfig dtclient.ProcessModuleConfig
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

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated")

		clt := fake.NewClient(dk)

		mockDTClient := dtclientmock.NewClient(t)
		expectedError := errors.New("API error")
		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).
			Return(nil, expectedError)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(context.Background(), dk)

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

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated")

		clt := fake.NewClient(dk) // No tenant secret

		mockDTClient := dtclientmock.NewClient(t)
		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).
			Return(&dtclient.ProcessModuleConfig{Properties: []dtclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(context.Background(), dk)

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
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: testUUID,
							Endpoints:  testCommunicationEndpoint,
						},
					},
				},
			},
		}

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated")

		clt := fake.NewClient(
			dk,
			clientSecret(dk.OneAgent().GetTenantSecret(), testNamespace, map[string][]byte{
				connectioninfo.TenantTokenKey: []byte(testTenantToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).
			Return(&dtclient.ProcessModuleConfig{Properties: []dtclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(context.Background(), dk)

		require.Error(t, err)
		require.Nil(t, result)

		mockDTClient.AssertExpectations(t)
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
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: testUUID,
							Endpoints:  testCommunicationEndpoint,
						},
					},
				},
			},
		}

		// Set condition as NOT outdated
		conditions.SetSecretCreated(dk.Conditions(), ConfigConditionType, "secret created")

		cachedPMCData, _ := json.Marshal(&dtclient.ProcessModuleConfig{Properties: []dtclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}})

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

		mockDTClient := dtclientmock.NewClient(t)
		// Should NOT call GetProcessModuleConfig when using cached data

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.preparePMC(context.Background(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)

		var pmConfig dtclient.ProcessModuleConfig
		err = json.Unmarshal(result, &pmConfig)
		require.NoError(t, err)

		// Verify API was not called
		mockDTClient.AssertNotCalled(t, "GetProcessModuleConfig")
	})
}

func TestGetCachedPMC(t *testing.T) {
	t.Run("returns nil when secret is outdated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
		}

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated")

		clt := fake.NewClient(dk)
		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.getCachedPMC(context.Background(), dk)

		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("returns cached PMC when available", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
		}

		conditions.SetSecretCreated(dk.Conditions(), ConfigConditionType, "secret created")

		cachedPMC := &dtclient.ProcessModuleConfig{Properties: []dtclient.ProcessModuleProperty{{Section: "test", Key: "test", Value: "test"}}}
		cachedPMCData, _ := json.Marshal(cachedPMC)

		sourceSecret := clientSecret(GetSourceConfigSecretName(dk.Name), testNamespace, map[string][]byte{
			pmc.InputFileName: cachedPMCData,
		})

		targetSecret := clientSecret(consts.BootstrapperInitSecretName, testNamespace, map[string][]byte{
			pmc.InputFileName: cachedPMCData,
		})

		clt := fake.NewClient(dk, sourceSecret, targetSecret)
		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.getCachedPMC(context.Background(), dk)

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("returns nil when source secret not found", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
		}

		conditions.SetSecretCreated(dk.Conditions(), ConfigConditionType, "secret created")

		clt := fake.NewClient(dk) // No secrets
		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.getCachedPMC(context.Background(), dk)

		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("returns nil when PMC data missing from source secret", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
		}

		conditions.SetSecretCreated(dk.Conditions(), ConfigConditionType, "secret created")

		sourceSecret := clientSecret(GetSourceConfigSecretName(dk.Name), testNamespace, map[string][]byte{
			"other-data": []byte("some-data"),
		})

		targetSecret := clientSecret(consts.BootstrapperInitSecretName, testNamespace, map[string][]byte{
			"other-data": []byte("some-data"),
		})

		clt := fake.NewClient(dk, sourceSecret, targetSecret)
		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.getCachedPMC(context.Background(), dk)

		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("returns error when PMC data is invalid", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
		}

		conditions.SetSecretCreated(dk.Conditions(), ConfigConditionType, "secret created")

		sourceSecret := clientSecret(GetSourceConfigSecretName(dk.Name), testNamespace, map[string][]byte{
			pmc.InputFileName: []byte("invalid-json-data"),
		})

		targetSecret := clientSecret(consts.BootstrapperInitSecretName, testNamespace, map[string][]byte{
			pmc.InputFileName: []byte("invalid-json-data"),
		})

		clt := fake.NewClient(dk, sourceSecret, targetSecret)
		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)

		result, err := secretGenerator.getCachedPMC(context.Background(), dk)

		require.Error(t, err)
		require.Nil(t, result)
	})
}
