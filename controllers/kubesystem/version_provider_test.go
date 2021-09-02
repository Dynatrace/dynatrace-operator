package kubesystem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	restclient "k8s.io/client-go/rest"
)

func TestDiscoveryVersionProvider(t *testing.T) {
	t.Run(`retrieves version info`, func(t *testing.T) {
		expect := version.Info{
			Major: "1",
			Minor: "20",
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			output, err := json.Marshal(expect)
			if err != nil {
				t.Errorf("unexpected encoding error: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(output)
		}))
		defer server.Close()

		versionProvider := NewVersionProvider(&restclient.Config{Host: server.URL})
		major, err := versionProvider.Major()

		assert.NoError(t, err)
		assert.Equal(t, major, "1")

		minor, err := versionProvider.Minor()

		assert.NoError(t, err)
		assert.Equal(t, minor, "20")
	})

	t.Run(`handles server errors`, func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(""))
		}))
		defer server.Close()

		versionProvider := NewVersionProvider(&restclient.Config{Host: server.URL})
		major, err := versionProvider.Major()

		assert.Error(t, err)
		assert.Equal(t, major, "")

		minor, err := versionProvider.Minor()

		assert.Error(t, err)
		assert.Equal(t, minor, "")
	})

	t.Run(`handles nil config`, func(t *testing.T) {
		versionProvider := NewVersionProvider(nil)
		major, err := versionProvider.Major()

		assert.EqualError(t, err, errorConfigIsNil)
		assert.Equal(t, major, "")

		minor, err := versionProvider.Minor()

		assert.EqualError(t, err, errorConfigIsNil)
		assert.Equal(t, minor, "")
	})

	t.Run(`handles invalid config`, func(t *testing.T) {
		versionProvider := NewVersionProvider(&restclient.Config{})
		major, err := versionProvider.Major()

		assert.Error(t, err)
		assert.Equal(t, major, "")

		minor, err := versionProvider.Minor()

		assert.Error(t, err)
		assert.Equal(t, minor, "")
	})
}
