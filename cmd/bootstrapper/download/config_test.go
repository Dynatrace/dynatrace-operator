package download

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToDTClientOptions(t *testing.T) {
	type testCase struct {
		title string
		in    Config
		out   []dynatrace.OptionV2
	}

	tests := []testCase{
		{
			title: "host group propagated",
			in:    Config{HostGroup: "host"},
			out:   []dynatrace.OptionV2{dynatrace.WithHostGroup("host")},
		},
		{
			title: "network zone propagated",
			in:    Config{NetworkZone: "network"},
			out:   []dynatrace.OptionV2{dynatrace.WithNetworkZone("network")},
		},
		{
			title: "proxy propagated",
			in:    Config{Proxy: "proxy", NoProxy: "no-proxy"},
			out:   []dynatrace.OptionV2{dynatrace.WithProxy("proxy", "no-proxy")},
		},
		{
			title: "skip cert check propagated",
			in:    Config{SkipCertCheck: true},
			out:   []dynatrace.OptionV2{dynatrace.WithSkipCertificateValidation(true)},
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
			out: []dynatrace.OptionV2{
				dynatrace.WithHostGroup("host"),
				dynatrace.WithNetworkZone("network"),
				dynatrace.WithProxy("proxy", "no-proxy"),
				dynatrace.WithSkipCertificateValidation(true),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			options := test.in.toDTClientOptionsV2()

			compareDTOptions(t, test.out, options)
		})
	}
}

func TestConfigFromFs(t *testing.T) {
	t.Run("missing file -> error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")

		config, err := configFromFs(inputDir)
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("not json -> error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		inputFile := filepath.Join(inputDir, InputFileName)
		os.WriteFile(inputFile, []byte("-------"), 0600)

		config, err := configFromFs(inputDir)
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("happy path", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")

		expected := testConfig(t)
		setupConfig(t, inputDir, expected)

		config, err := configFromFs(inputDir)
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, expected, *config)
	})
}

func testConfig(t *testing.T) Config {
	t.Helper()

	return Config{
		URL:           "url",
		APIToken:      "token",
		SkipCertCheck: true,
		Proxy:         "proxy",
		NoProxy:       "no-proxy",
		NetworkZone:   "network",
		HostGroup:     "host",
	}
}

func setupConfig(t *testing.T, inputDir string, config Config) {
	t.Helper()

	raw, err := json.Marshal(config)
	require.NoError(t, err)

	os.Mkdir(inputDir, os.ModePerm)
	err = os.WriteFile(filepath.Join(inputDir, InputFileName), raw, 0600)
	require.NoError(t, err)
}

func compareDTOptions(t *testing.T, opts1 []dynatrace.OptionV2, opts2 []dynatrace.OptionV2) {
	require.Len(t, opts1, len(opts2))
	for i := range opts1 {
		expected := getNameOfCalledFunc(t, opts1[i])
		actual := getNameOfCalledFunc(t, opts2[i])
		assert.Equal(t, expected, actual)
	}
}

func getNameOfCalledFunc(t *testing.T, option dynatrace.OptionV2) string {
	t.Helper()

	funcPath := strings.Split(runtime.FuncForPC(reflect.ValueOf(option).Pointer()).Name(), ".")

	return funcPath[len(funcPath)-2]
}
