package dynatrace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	oneAgentImageURL    = "/v1/deployment/image/agent/oneAgent/latest"
	codeModulesImageURL = "/v1/deployment/image/agent/codeModules/latest"
	activeGateImageURL  = "/v1/deployment/image/gateway/latest"
)

var latestOneAgentImageResponse = &LatestImageInfo{
	Source: "dt.oneAgent/test",
	Tag:    "1.xxx",
}

var latestActiveGateImageResponse = &LatestImageInfo{
	Source: "dt.activeGate/test",
	Tag:    "1.xxx",
}

var latestCodeModulesImageResponse = &LatestImageInfo{
	Source: "dt.codeModules/test",
	Tag:    "1.xxx",
}

func TestGetLatestImage(t *testing.T) {
	ctx := context.Background()

	t.Run("GetLatestOneAgentImage works", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(oneAgentImageURL, latestOneAgentImageResponse), "")
		defer dynatraceServer.Close()

		latestImageInfo, err := dynatraceClient.GetLatestOneAgentImage(ctx)
		require.NoError(t, err)
		assert.NotNil(t, latestImageInfo)

		assert.Equal(t, latestImageInfo.Source, latestOneAgentImageResponse.Source)
		assert.Contains(t, latestImageInfo.Tag, latestOneAgentImageResponse.Tag)
	})
	t.Run("GetLatestActiveGateImage works", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(activeGateImageURL, latestActiveGateImageResponse), "")
		defer dynatraceServer.Close()

		latestImageInfo, err := dynatraceClient.GetLatestActiveGateImage(ctx)
		require.NoError(t, err)
		assert.NotNil(t, latestImageInfo)

		assert.Equal(t, latestImageInfo.Source, latestActiveGateImageResponse.Source)
		assert.Contains(t, latestImageInfo.Tag, latestActiveGateImageResponse.Tag)
	})
	t.Run("GetLatestCodeModulesImage works", func(t *testing.T) {
		dynatraceServer, dynatraceClient := createTestDynatraceServer(t, connectionInfoServerHandler(codeModulesImageURL, latestCodeModulesImageResponse), "")
		defer dynatraceServer.Close()

		latestImageInfo, err := dynatraceClient.GetLatestCodeModulesImage(ctx)
		require.NoError(t, err)
		assert.NotNil(t, latestImageInfo)

		assert.Equal(t, latestImageInfo.Source, latestCodeModulesImageResponse.Source)
		assert.Contains(t, latestImageInfo.Tag, latestCodeModulesImageResponse.Tag)
	})
}

func TestGetLatestImageFailure(t *testing.T) {
	ctx := context.Background()

	t.Run("GetLatestOneAgentImage handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(oneAgentImageURL), "")
		defer faultyDynatraceServer.Close()

		latestImageInfo, err := faultyDynatraceClient.GetLatestOneAgentImage(ctx)
		require.Error(t, err)
		assert.Nil(t, latestImageInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
	t.Run("GetLatestActiveGateImage handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(activeGateImageURL), "")
		defer faultyDynatraceServer.Close()

		latestImageInfo, err := faultyDynatraceClient.GetLatestActiveGateImage(ctx)
		require.Error(t, err)
		assert.Nil(t, latestImageInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
	t.Run("GetLatestCodeModulesImage handle internal server error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantInternalServerError(codeModulesImageURL), "")
		defer faultyDynatraceServer.Close()

		latestImageInfo, err := faultyDynatraceClient.GetLatestCodeModulesImage(ctx)
		require.Error(t, err)
		assert.Nil(t, latestImageInfo)

		assert.Equal(t, "dynatrace server error 500: error retrieving tenant info", err.Error())
	})
	t.Run("GetLatestOneAgentImage handle malformed json error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJSON(oneAgentImageURL), "")
		defer faultyDynatraceServer.Close()

		latestImageInfo, err := faultyDynatraceClient.GetLatestOneAgentImage(ctx)
		require.Error(t, err)
		assert.Nil(t, latestImageInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
	t.Run("GetLatestActiveGateImage handle malformed json error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJSON(activeGateImageURL), "")
		defer faultyDynatraceServer.Close()

		latestImageInfo, err := faultyDynatraceClient.GetLatestActiveGateImage(ctx)
		require.Error(t, err)
		assert.Nil(t, latestImageInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
	t.Run("GetLatestCodeModulesImage handle malformed json error", func(t *testing.T) {
		faultyDynatraceServer, faultyDynatraceClient := createTestDynatraceServer(t, tenantMalformedJSON(codeModulesImageURL), "")
		defer faultyDynatraceServer.Close()

		latestImageInfo, err := faultyDynatraceClient.GetLatestCodeModulesImage(ctx)
		require.Error(t, err)
		assert.Nil(t, latestImageInfo)

		assert.Equal(t, "invalid character 'h' in literal true (expecting 'r')", err.Error())
	})
}
