package dynatrace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynatraceClient_CreateLogMonitoringSetting(t *testing.T) {
	ctx := context.Background()

	t.Run("create settings logmonitoring settings with default rule matchers", func(t *testing.T) {
		// arrange
		dynakube := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{},
			},
		}

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateLogMonitoringSetting(ctx, testScope, testName, dynakube.LogMonitoring().IngestRuleMatchers)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Len(t, actual, len(testObjectID))
		assert.Equal(t, testObjectID, actual)
	})
	t.Run("create settings logmonitoring settings with custom rule matchers", func(t *testing.T) {
		// arrange
		dynakube := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{
					IngestRuleMatchers: []logmonitoring.IngestRuleMatchers{
						{
							Attribute: "test-attribute",
							Values: []string{
								"test-value",
							},
						},
					},
				},
			},
		}

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateLogMonitoringSetting(ctx, testScope, testName, dynakube.LogMonitoring().IngestRuleMatchers)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Len(t, actual, len(testObjectID))
		assert.Equal(t, testObjectID, actual)
	})
}
