package istio

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/stretchr/testify/assert"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultTestNamespace = "dynatrace"
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

	vs := buildVirtualService("dt-vs", "ENVIRONMENTID.live.dynatrace.com", "https", 443)
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
