package image

import (
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestOneAgentImage(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().Execute(new(LatestImageInfo)).Run(injectResponse(LatestImageInfo{Source: "oneagent", Tag: "1.2.3"})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, LatestOneAgentImagePath).Return(request).Once()

		client := NewClient(apiClient)
		latestImageInfo, err := client.LatestOneAgentImage(ctx)
		require.NoError(t, err)
		assert.Equal(t, "oneagent", latestImageInfo.Source)
		assert.Equal(t, "1.2.3", latestImageInfo.Tag)
	})
}

func TestLatestCodeModulesImage(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().Execute(new(LatestImageInfo)).Run(injectResponse(LatestImageInfo{Source: "codemodules", Tag: "1.2.3"})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, LatestCodeModulesImagePath).Return(request).Once()

		client := NewClient(apiClient)
		latestImageInfo, err := client.LatestCodeModulesImage(ctx)
		require.NoError(t, err)
		assert.Equal(t, "codemodules", latestImageInfo.Source)
		assert.Equal(t, "1.2.3", latestImageInfo.Tag)
	})
}

func TestLatestActiveGateImage(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().Execute(new(LatestImageInfo)).Run(injectResponse(LatestImageInfo{Source: "activegate", Tag: "1.2.3"})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, LatestActiveGateImagePath).Return(request).Once()

		client := NewClient(apiClient)
		latestImageInfo, err := client.LatestActiveGateImage(ctx)
		require.NoError(t, err)
		assert.Equal(t, "activegate", latestImageInfo.Source)
		assert.Equal(t, "1.2.3", latestImageInfo.Tag)
	})
}

func injectResponse[T any](resp T) func(any) {
	return func(arg any) {
		if target, ok := arg.(*T); ok {
			*target = resp
		}
	}
}
