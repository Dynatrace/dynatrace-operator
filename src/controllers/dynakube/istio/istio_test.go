package istio

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

func TestIstioEnabled(t *testing.T) {
	server := initMockServer(true)
	defer server.Close()
	cfg := &restclient.Config{Host: server.URL}
	r, e := CheckIstioInstalled(cfg)
	if r != true {
		t.Error(e)
	}
}

func TestIstioDisabled(t *testing.T) {
	server := initMockServer(false)
	defer server.Close()
	cfg := &restclient.Config{Host: server.URL}
	r, e := CheckIstioInstalled(cfg)
	if r != false && e == nil {
		t.Errorf("expected false, got true, %v", e)
	}
}

func TestIstioWrongConfig(t *testing.T) {
	server := initMockServer(false)
	defer server.Close()
	cfg := &restclient.Config{Host: "localhost:1000"}

	r, e := CheckIstioInstalled(cfg)
	if r == false && e != nil { // only true success case
		t.Logf("got false and error: %v", e)
	} else {
		t.Error("got true, expected false with error")
	}
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
