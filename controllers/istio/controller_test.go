package istio

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	DefaultTestNamespace = "dynatrace"

	testVirtualServiceName     = "dt-vs"
	testVirtualServiceHost     = "ENVIRONMENTID.live.dynatrace.com"
	testVirtualServiceProtocol = "https"
	testVirtualServicePort     = 443

	testApiPath = "/path"
	testVersion = "apps/v1"
)

func TestIstioClient_CreateIstioObjects(t *testing.T) {
	buffer := bytes.NewBufferString("{\"apiVersion\":\"networking.istio.io/v1alpha3\",\"kind\":\"VirtualService\",\"metadata\":{\"clusterName\":\"\",\"creationTimestamp\":\"2018-11-26T03:19:57Z\",\"generation\":1,\"name\":\"test-virtual-service\",\"namespace\":\"istio-system\",\"resourceVersion\":\"1297970\",\"selfLink\":\"/apis/networking.istio.io/v1alpha3/namespaces/istio-system/virtualservices/test-virtual-service\",\"uid\":\"266fdacc-f12a-11e8-9e1d-42010a8000ff\"},\"spec\":{\"gateways\":[\"test-gateway\"],\"hosts\":[\"*\"],\"http\":[{\"match\":[{\"uri\":{\"prefix\":\"/\"}}],\"route\":[{\"destination\":{\"host\":\"test-service\",\"port\":{\"number\":8080}}}],\"timeout\":\"10s\"}]}}\n")

	vs := istiov1alpha3.VirtualService{}
	assert.NoError(t, json.Unmarshal(buffer.Bytes(), &vs))

	ic := fakeistio.NewSimpleClientset(&vs)

	vsList, err := ic.NetworkingV1alpha3().VirtualServices("istio-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", DefaultTestNamespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestIstioClient_BuildDynatraceVirtualService(t *testing.T) {
	err := os.Setenv("POD_NAMESPACE", DefaultTestNamespace)
	if err != nil {
		t.Error("Failed to set environment variable")
	}

	vs := buildVirtualService(testVirtualServiceName, DefaultTestNamespace, testVirtualServiceHost, testVirtualServiceProtocol, testVirtualServicePort)
	ic := fakeistio.NewSimpleClientset(vs)
	vsList, err := ic.NetworkingV1alpha3().VirtualServices(DefaultTestNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", DefaultTestNamespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestController_ReconcileIstio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(reconcileTestHandler))
	defer server.Close()

	serverUrl, err := url.Parse(server.URL)
	require.NoError(t, err)

	port, err := strconv.ParseUint(serverUrl.Port(), 10, 32)
	require.NoError(t, err)

	virtualService := buildVirtualService(testVirtualServiceName, DefaultTestNamespace, "localhost", serverUrl.Scheme, uint32(port))
	instance := &dynatracev1.DynaKube{}
	controller := Controller{
		istioClient: fakeistio.NewSimpleClientset(virtualService),
		scheme:      scheme.Scheme,
		logger:      logger.NewDTLogger(),
		config: &rest.Config{
			Host:    server.URL,
			APIPath: testApiPath,
		},
	}

	updated, err := controller.ReconcileIstio(instance)

	assert.NoError(t, err)
	assert.False(t, updated)
}

func reconcileTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/apis" {
		sendApiGroupList(w)
	} else {
		sendApiVersions(w)
	}
}

func sendApiVersions(w http.ResponseWriter) {
	versions := metav1.APIVersions{
		Versions: []string{testVersion},
	}
	sendData(versions, w)
}

func sendApiGroupList(w http.ResponseWriter) {
	apiGroupList := metav1.APIGroupList{
		Groups: []metav1.APIGroup{
			{
				Name: istioGVRName,
			},
		},
	}
	sendData(apiGroupList, w)
}

func sendData(i interface{}, w http.ResponseWriter) {
	data, err := json.Marshal(i)

	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
	}

	w.WriteHeader(200)
	_, _ = w.Write(data)
}
