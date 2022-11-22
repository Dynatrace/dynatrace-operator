package manifests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func httpGetResponseReader(url string) (io.Reader, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Response status code was not 200(StatusOK): " + strconv.Itoa(response.StatusCode))
	}

	manifestBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(manifestBytes), nil
}

func InstallFromUrls(yamlUrls []string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		for _, yamlUrl := range yamlUrls {
			installFromSingleUrl(t, ctx, environmentConfig, yamlUrl)
		}

		return ctx
	}
}

func installFromSingleUrl(t *testing.T, ctx context.Context, environmentConfig *envconf.Config, yamlUrl string) {
	manifestReader, err := httpGetResponseReader(yamlUrl)
	require.NoError(t, err, "could not fetch release yaml")

	resources := environmentConfig.Client().Resources()
	require.NoError(t, decoder.DecodeEach(ctx, manifestReader, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), func(err error) bool {
		// Ignore if the resource already exists
		return k8serrors.IsAlreadyExists(err)
	})))
}

func InstallFromFiles(paths []string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		for _, path := range paths {
			installFromSingleFile(t, ctx, environmentConfig, path)
		}
		return ctx
	}
}

func InstallFromFile(path string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		installFromSingleFile(t, ctx, environmentConfig, path)
		return ctx
	}
}

func installFromSingleFile(t *testing.T, ctx context.Context, environmentConfig *envconf.Config, path string) {
	manifest, err := os.Open(path)
	defer func() { require.NoError(t, manifest.Close()) }()
	require.NoError(t, err)

	resources := environmentConfig.Client().Resources()
	require.NoError(t, decoder.DecodeEach(ctx, manifest, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), func(err error) bool {
		// Ignore if the resource already exists
		return k8serrors.IsAlreadyExists(err)
	})))
}
