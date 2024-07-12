package edgeconnect

import (
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestApiServer(t *testing.T) {
	t.Run(`happy apiServer`, func(t *testing.T) {
		for _, suffix := range allowedSuffix {
			ec := &edgeconnect.EdgeConnect{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: edgeconnect.EdgeConnectSpec{
					ApiServer: "tenantid" + suffix,
					OAuth: edgeconnect.OAuthSpec{
						ClientSecret: "secret",
						Endpoint:     "endpoint",
						Resource:     "resource",
					},
					ServiceAccountName: testServiceAccountName,
				},
			}
			assertAllowedResponse(t, ec, prepareTestServiceAccount(testServiceAccountName, testNamespace))
		}
	})

	t.Run(`invalid apiServer (missing tenant)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: allowedSuffix[0],
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run(`invalid apiServer (wrong suffix)`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorInvalidApiServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "doma.in",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})
}

func assertAllowedResponse(t *testing.T, ec *edgeconnect.EdgeConnect, other ...client.Object) {
	response := handleRequest(t, ec, other...)
	assert.True(t, response.Allowed, response.Result.Message)
	assert.Empty(t, response.Warnings)
}

func assertDeniedResponse(t *testing.T, errMessages []string, ec *edgeconnect.EdgeConnect, other ...client.Object) {
	response := handleRequest(t, ec, other...)
	assert.False(t, response.Allowed)

	for _, errMsg := range errMessages {
		assert.Contains(t, response.Result.Message, errMsg)
	}
}

func handleRequest(t *testing.T, ec *edgeconnect.EdgeConnect, other ...client.Object) admission.Response {
	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &edgeconnectValidator{
		clt:       clt,
		apiReader: clt,
		cfg:       &rest.Config{},
	}

	data, err := json.Marshal(*ec)
	require.NoError(t, err)

	return validator.Handle(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name:      testName,
			Namespace: testNamespace,
			Object:    runtime.RawExtension{Raw: data},
		},
	})
}
