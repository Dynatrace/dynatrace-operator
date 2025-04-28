package download

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToDTClientOptions(t *testing.T) {
	type testCase struct {
		title string
		in    Config
		out   []dtclient.Option
	}

	tests := []testCase{
		{
			title: "host group propagated",
			in:    Config{HostGroup: "host"},
			out:   []dtclient.Option{dtclient.HostGroup("host")},
		},
		{
			title: "network zone propagated",
			in:    Config{NetworkZone: "network"},
			out:   []dtclient.Option{dtclient.NetworkZone("network")},
		},
		{
			title: "proxy propagated",
			in:    Config{Proxy: "proxy", NoProxy: "no-proxy"},
			out:   []dtclient.Option{dtclient.Proxy("proxy", "no-proxy")},
		},
		{
			title: "skip cert check propagated",
			in:    Config{SkipCertCheck: true},
			out:   []dtclient.Option{dtclient.SkipCertificateValidation(true)},
		},
		{
			title: "everything propagated",
			in: Config{
				HostGroup:     "host",
				NetworkZone:   "network",
				Proxy:         "proxy",
				NoProxy:       "no-proxy",
				SkipCertCheck: true,
			},
			out: []dtclient.Option{
				dtclient.HostGroup("host"),
				dtclient.NetworkZone("network"),
				dtclient.Proxy("proxy", "no-proxy"),
				dtclient.SkipCertificateValidation(true),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			options := test.in.toDTClientOptions()

			compareDTOptions(t, test.out, options)
		})
	}
}

func TestConfigFromFs(t *testing.T) {
	inputDir := "in"
	inputFile := filepath.Join(inputDir, InputFileName)

	t.Run("missing file -> error", func(t *testing.T) {
		fs := afero.Afero{Fs: afero.NewMemMapFs()}

		config, err := configFromFs(fs, inputDir)
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("not json -> error", func(t *testing.T) {
		fs := afero.Afero{Fs: afero.NewMemMapFs()}
		fs.WriteFile(inputFile, []byte("-------"), os.ModePerm)

		config, err := configFromFs(fs, inputDir)
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("happy path", func(t *testing.T) {
		fs := afero.Afero{Fs: afero.NewMemMapFs()}

		expected := testConfig(t)
		setupConfig(t, &fs, inputDir, expected)

		config, err := configFromFs(fs, inputDir)
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, expected, *config)
	})
}

func testConfig(t *testing.T) Config {
	t.Helper()

	return Config{
		URL:           "url",
		ApiToken:      "token",
		SkipCertCheck: true,
		Proxy:         "proxy",
		NoProxy:       "no-proxy",
		NetworkZone:   "network",
		HostGroup:     "host",
	}
}

func setupConfig(t *testing.T, fs *afero.Afero, inputDir string, config Config) {
	t.Helper()

	raw, err := json.Marshal(config)
	require.NoError(t, err)

	fs.WriteFile(filepath.Join(inputDir, InputFileName), raw, os.ModePerm)
}

func compareDTOptions(t *testing.T, opts1 []dtclient.Option, opts2 []dtclient.Option) {
	require.Len(t, opts1, len(opts2))
	for i := range opts1 {
		expected := getNameOfCalledFunc(t, opts1[i])
		actual := getNameOfCalledFunc(t, opts2[i])
		assert.Equal(t, expected, actual)
	}
}

func getNameOfCalledFunc(t *testing.T, option dtclient.Option) string {
	t.Helper()

	funcPath := strings.Split(runtime.FuncForPC(reflect.ValueOf(option).Pointer()).Name(), ".")

	return funcPath[len(funcPath)-2]
}
