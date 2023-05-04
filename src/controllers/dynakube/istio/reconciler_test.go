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

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
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
	err := os.Setenv(kubeobjects.EnvPodNamespace, DefaultTestNamespace)
	if err != nil {
		t.Error("Failed to set environment variable")
	}

	vs := buildVirtualService(buildObjectMeta(testVirtualServiceName, DefaultTestNamespace), testVirtualServiceHost, testVirtualServiceProtocol, testVirtualServicePort)
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
	testReconcileIstio(t, true)
}

func TestController_ReconcileIstioGracefullyFail(t *testing.T) {
	testReconcileIstio(t, false)
}

func testReconcileIstio(t *testing.T, enableIstioGVR bool) {
	server := httptest.NewServer(createReconcileTestHandler(true))
	defer server.Close()

	serverUrl, err := url.Parse(server.URL)
	require.NoError(t, err)

	port, err := strconv.ParseUint(serverUrl.Port(), 10, 32)
	require.NoError(t, err)

	virtualService := buildVirtualService(buildObjectMeta(testVirtualServiceName, DefaultTestNamespace), "localhost", serverUrl.Scheme, uint32(port))
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: serverUrl.String(),
		},
	}
	reconciler := Reconciler{
		istioClient: fakeistio.NewSimpleClientset(virtualService),
		scheme:      scheme.Scheme,
		config: &rest.Config{
			Host:    server.URL,
			APIPath: testApiPath,
		},
	}

	updated, err := reconciler.Reconcile(instance, []dtclient.CommunicationHost{})

	assert.NoError(t, err)
	assert.False(t, updated)
}

func TestController_ReconcileIstio2(t *testing.T) {
	server := httptest.NewServer(createReconcileTestHandler(false))
	defer server.Close()

	serverUrl, err := url.Parse(server.URL)
	require.NoError(t, err)

	port, err := strconv.ParseUint(serverUrl.Port(), 10, 32)
	require.NoError(t, err)

	virtualService := buildVirtualService(buildObjectMeta(testVirtualServiceName, DefaultTestNamespace), "localhost", serverUrl.Scheme, uint32(port))
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: serverUrl.String(),
		},
	}
	reconciler := Reconciler{
		istioClient: fakeistio.NewSimpleClientset(virtualService),
		scheme:      scheme.Scheme,
		config: &rest.Config{
			Host:    server.URL,
			APIPath: testApiPath,
		},
	}

	updated, err := reconciler.Reconcile(instance, []dtclient.CommunicationHost{})

	assert.NoError(t, err)
	assert.False(t, updated)
}

func createReconcileTestHandler(enableIstioGVR bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/apis" {
			apiGroupList := []metav1.APIGroup{}

			if enableIstioGVR {
				apiGroupList = append(apiGroupList, metav1.APIGroup{Name: IstioGVRName})
			}

			sendApiGroupList(w, apiGroupList)
		} else {
			sendApiVersions(w)
		}
	}
}

func sendApiVersions(w http.ResponseWriter) {
	versions := metav1.APIVersions{
		Versions: []string{testVersion},
	}
	sendData(versions, w)
}

func sendApiGroupList(w http.ResponseWriter, apiGroups []metav1.APIGroup) {
	apiGroupList := metav1.APIGroupList{
		Groups: apiGroups,
	}
	sendData(apiGroupList, w)
}

func sendData(i interface{}, w http.ResponseWriter) {
	data, err := json.Marshal(i)

	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, _ = w.Write(data)
}
