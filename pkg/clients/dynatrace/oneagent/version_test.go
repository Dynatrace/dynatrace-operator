package oneagent

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	agentResponse          = `zip-content`
	versionedAgentResponse = `zip-content-1.2.3`
)

func TestGetLatest(t *testing.T) {
	args := GetParams{
		OS:            installer.OSUnix,
		InstallerType: installer.TypePaaS,
		Flavor:        arch.FlavorMultidistro,
		Technologies:  nil,
		SkipMetadata:  false,
	}

	setupClient := func(t *testing.T, response []byte, rawErr error) (*ClientImpl, *os.File) {
		file, err := os.CreateTemp(t.TempDir(), "installer")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, file.Close()) })

		hash := sha256.New()
		multiWriter := io.MultiWriter(file, hash)

		req := coremock.NewRequest(t)
		req.EXPECT().WithPath([]string{args.OS, args.InstallerType, "latest"}).Return(req).Once()
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithRawQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithHeader("Accept", "application/octet-stream").Return(req).Once()
		req.EXPECT().ExecuteWriter(multiWriter).Run(func(writer io.Writer) {
			_, copyErr := io.Copy(writer, bytes.NewReader(response))
			require.NoError(t, copyErr)
		}).Return(nil, rawErr).Once()

		coreClient := coremock.NewClient(t)
		coreClient.EXPECT().GET(t.Context(), agentDeploymentPath).Return(req).Once()

		return NewClient(coreClient, "", ""), file
	}

	t.Run("file download successful", func(t *testing.T) {
		oaClient, file := setupClient(t, []byte(agentResponse), nil)
		err := oaClient.GetLatest(t.Context(), args, file)
		require.NoError(t, err)

		resp, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		assert.Equal(t, agentResponse, string(resp))
	})

	t.Run("agent not found error", func(t *testing.T) {
		oaClient, file := setupClient(t, nil, &core.HTTPError{StatusCode: 404, Message: "Not found"})
		err := oaClient.GetLatest(t.Context(), args, file)
		require.Error(t, err)
	})

	t.Run("missing params", func(t *testing.T) {
		require.ErrorIs(t, (&ClientImpl{}).GetLatest(t.Context(), GetParams{InstallerType: installer.TypePaaS}, nil), errEmptyOS)
		require.ErrorIs(t, (&ClientImpl{}).GetLatest(t.Context(), GetParams{OS: installer.OSUnix}, nil), errEmptyInstallerType)
	})
}

func TestGet(t *testing.T) {
	args := GetParams{
		OS:            installer.OSUnix,
		InstallerType: installer.TypePaaS,
		Version:       "1.2.3",
		Flavor:        "",
		Technologies:  nil,
		SkipMetadata:  false,
	}

	setupClient := func(t *testing.T, response []byte, rawErr error) (*ClientImpl, *os.File) {
		file, err := os.CreateTemp(t.TempDir(), "installer")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, file.Close()) })

		hash := sha256.New()
		multiWriter := io.MultiWriter(file, hash)

		req := coremock.NewRequest(t)
		req.EXPECT().WithPath([]string{args.OS, args.InstallerType, "version", args.Version}).Return(req).Once()
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithRawQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithHeader(mock.Anything, mock.Anything).Return(req).Once()
		req.EXPECT().ExecuteWriter(multiWriter).Run(func(writer io.Writer) {
			_, copyErr := io.Copy(writer, bytes.NewReader(response))
			require.NoError(t, copyErr)
		}).Return(nil, rawErr).Once()

		coreClient := coremock.NewClient(t)
		coreClient.EXPECT().GET(t.Context(), agentDeploymentPath).Return(req).Once()

		return NewClient(coreClient, "", ""), file
	}

	t.Run("handle response correctly", func(t *testing.T) {
		oaClient, file := setupClient(t, []byte(versionedAgentResponse), nil)
		err := oaClient.Get(t.Context(), args, file)
		require.NoError(t, err)

		resp, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		assert.Equal(t, versionedAgentResponse, string(resp))
	})

	t.Run("handle server error", func(t *testing.T) {
		oaClient, file := setupClient(t, nil, &core.HTTPError{StatusCode: 404, Message: "Not found"})
		err := oaClient.Get(t.Context(), args, file)

		require.True(t, core.IsNotFound(err))
	})

	t.Run("missing params", func(t *testing.T) {
		require.ErrorIs(t, (&ClientImpl{}).Get(t.Context(), GetParams{InstallerType: installer.TypePaaS}, nil), errEmptyOS)
		require.ErrorIs(t, (&ClientImpl{}).Get(t.Context(), GetParams{OS: installer.OSUnix}, nil), errEmptyInstallerType)
	})
}

func TestGetVersions(t *testing.T) {
	args := GetParams{
		OS:            installer.OSUnix,
		InstallerType: installer.TypePaaS,
		Flavor:        "",
	}
	responseString := []string{"1.123.1", "1.123.2", "1.123.3", "1.123.4"}

	setupClient := func(t *testing.T, execErr error) *ClientImpl {
		var resp versionsResponse

		req := coremock.NewRequest(t)
		req.EXPECT().WithPath([]string{"versions", args.OS, args.InstallerType}).Return(req).Once()
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().Execute(&resp).Run(func(model any) {
			if execErr == nil {
				resp := model.(*versionsResponse)
				resp.AvailableVersions = responseString
			}
		}).Return(execErr).Once()

		coreClient := coremock.NewClient(t)
		coreClient.EXPECT().GET(t.Context(), agentDeploymentPath).Return(req).Once()

		return NewClient(coreClient, "", "")
	}

	t.Run("handle response correctly", func(t *testing.T) {
		oaClient := setupClient(t, nil)
		availableVersions, err := oaClient.GetVersions(t.Context(), args)

		require.NoError(t, err)
		assert.Len(t, availableVersions, 4)
		assert.Contains(t, availableVersions, "1.123.1")
		assert.Contains(t, availableVersions, "1.123.2")
		assert.Contains(t, availableVersions, "1.123.3")
		assert.Contains(t, availableVersions, "1.123.4")
	})

	t.Run("handle server error", func(t *testing.T) {
		oaClient := setupClient(t, &core.HTTPError{StatusCode: 400, Message: "test-error"})
		availableVersions, err := oaClient.GetVersions(t.Context(), args)

		require.Empty(t, availableVersions)
		require.Error(t, err)
		require.True(t, core.IsBadRequest(err))
	})

	t.Run("missing params", func(t *testing.T) {
		_, err := (&ClientImpl{}).GetVersions(t.Context(), GetParams{InstallerType: installer.TypePaaS})
		require.ErrorIs(t, err, errEmptyOS)
		_, err = (&ClientImpl{}).GetVersions(t.Context(), GetParams{OS: installer.OSUnix})
		require.ErrorIs(t, err, errEmptyInstallerType)
	})
}
