//go:build e2e

package manifests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallFromFile(path string, options ...decoder.DecodeOption) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		kubernetesManifest, err := os.Open(path)
		defer func() { require.NoError(t, kubernetesManifest.Close()) }()
		require.NoError(t, err)

		resources := envConfig.Client().Resources()
		require.NoError(t, decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), k8serrors.IsAlreadyExists), options...))

		return ctx
	}
}

func UninstallFromFile(path string, options ...decoder.DecodeOption) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		kubernetesManifest, err := os.Open(path)
		defer func() { require.NoError(t, kubernetesManifest.Close()) }()
		require.NoError(t, err)

		resources := envConfig.Client().Resources()
		require.NoError(t, decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.DeleteHandler(resources), k8serrors.IsNotFound), options...))

		return ctx
	}
}

func InstallFromUrls(yamlUrls []string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		for _, yamlUrl := range yamlUrls {
			installFromSingleUrl(t, ctx, envConfig, yamlUrl)
		}

		return ctx
	}
}

func UninstallFromUrls(yamlUrls []string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		for _, yamlUrl := range yamlUrls {
			uninstallFromSingleUrl(t, ctx, envConfig, yamlUrl)
		}

		return ctx
	}
}

func installFromSingleUrl(t *testing.T, ctx context.Context, envConfig *envconf.Config, yamlUrl string) {
	manifestReader, err := httpGetResponseReader(yamlUrl)
	require.NoError(t, err, "could not fetch yaml")

	resources := envConfig.Client().Resources()
	require.NoError(t, decoder.DecodeEach(ctx, manifestReader, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), k8serrors.IsAlreadyExists)))
}

func uninstallFromSingleUrl(t *testing.T, ctx context.Context, envConfig *envconf.Config, yamlUrl string) {
	manifestReader, err := httpGetResponseReader(yamlUrl)
	require.NoError(t, err, "could not fetch yaml")

	resources := envConfig.Client().Resources()
	require.NoError(t, decoder.DecodeEach(ctx, manifestReader, decoder.IgnoreErrorHandler(decoder.DeleteHandler(resources), k8serrors.IsNotFound)))
}

func httpGetResponseReader(url string) (io.Reader, error) {
	response, err := http.Get(url) // nolint:gosec,bodyclose // G107: Potential HTTP request made with variable url - fine, same applies to naked `http.Get(url)`
	if err != nil {
		return nil, err
	}
	defer dtclient.CloseBodyAfterRequest(response)

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Response status code was not 200(StatusOK): " + strconv.Itoa(response.StatusCode))
	}

	manifestBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(manifestBytes), nil
}
