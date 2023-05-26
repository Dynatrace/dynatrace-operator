package istio

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

func TestIstioEnabled(t *testing.T) {
	server := initMockServer(true)
	defer server.Close()
	config := &restclient.Config{Host: server.URL}

	isInstalled, err := CheckIstioInstalled(config)
	require.True(t, isInstalled, err)
}

func TestIstioDisabled(t *testing.T) {
	server := initMockServer(false)
	defer server.Close()
	config := &restclient.Config{Host: server.URL}

	isInstalled, err := CheckIstioInstalled(config)
	require.False(t, isInstalled)
	require.Nil(t, err)
}

func TestIstioWrongConfig(t *testing.T) {
	server := initMockServer(false)
	defer server.Close()
	config := &restclient.Config{Host: "localhost:1000"}

	isInstalled, err := CheckIstioInstalled(config)
	require.False(t, isInstalled)
	require.NotNil(t, err)
}

func initMockServer(enableIstioGVR bool) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/apis/networking.istio.io/v1alpha3" {
			if !enableIstioGVR {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			apiResourceList := metav1.APIResourceList{
				GroupVersion: "networking.istio.io/v1alpha3",
				APIResources: []metav1.APIResource{
					{Name: "serviceentries", Namespaced: true, Kind: "ServiceEntry", Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
					{Name: "virtualservices", Namespaced: true, Kind: "VirtualService", Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
				},
			}

			sendData(apiResourceList, w)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	return server
}
