package utils

import (
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExtractToken(t *testing.T) {
	{
		secret := corev1.Secret{}
		_, err := extractToken(&secret, "test_token")
		assert.EqualError(t, err, "missing token test_token")
	}
	{
		data := map[string][]byte{}
		data["test_token"] = []byte("")
		secret := corev1.Secret{Data: data}
		token, err := extractToken(&secret, "test_token")
		assert.NoError(t, err)
		assert.Equal(t, token, "")
	}
	{
		data := map[string][]byte{}
		data["test_token"] = []byte("dynatrace_test_token")
		secret := corev1.Secret{Data: data}
		token, err := extractToken(&secret, "test_token")
		assert.NoError(t, err)
		assert.Equal(t, token, "dynatrace_test_token")
	}
	{
		data := map[string][]byte{}
		data["test_token"] = []byte("dynatrace_test_token \t \n")
		data["test_token_2"] = []byte("\t\n   dynatrace_test_token_2")
		secret := corev1.Secret{Data: data}

		token, err := extractToken(&secret, "test_token")
		assert.NoError(t, err)
		assert.Equal(t, token, "dynatrace_test_token")

		token2, err := extractToken(&secret, "test_token_2")
		assert.NoError(t, err)
		assert.Equal(t, token2, "dynatrace_test_token_2")
	}
}

func TestBuildDynatraceClient(t *testing.T) {
	namespace := "dynatrace"
	dynaKube := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "dynakube", Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: "custom-token",
			OneAgent: dynatracev1alpha1.OneAgentSpec{
				Enabled: true,
			},
		},
	}

	{
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-token", Namespace: namespace},
				Type:       corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"paasToken": []byte("42"),
					"apiToken":  []byte("43"),
				},
			},
		).Build()

		_, err := BuildDynatraceClient(fakeClient, dynaKube, true, true)
		assert.NoError(t, err)
	}

	{
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		_, err := BuildDynatraceClient(fakeClient, dynaKube, true, true)
		assert.Error(t, err)
	}

	{
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-token", Namespace: namespace},
				Type:       corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"paasToken": []byte("42"),
				},
			},
		).Build()
		_, err := BuildDynatraceClient(fakeClient, dynaKube, true, true)
		assert.Error(t, err)
	}
}

// GetDeployment returns the Deployment object who is the owner of this pod.
func TestGetDeployment(t *testing.T) {
	const ns = "dynatrace"

	os.Setenv("POD_NAME", "mypod")
	trueVar := true

	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypod",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "myreplicaset", Controller: &trueVar},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myreplicaset",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "mydeployment", Controller: &trueVar},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydeployment",
				Namespace: ns,
			},
		}).Build()

	deploy, err := GetDeployment(fakeClient, "dynatrace")
	require.NoError(t, err)
	assert.Equal(t, "mydeployment", deploy.Name)
	assert.Equal(t, "dynatrace", deploy.Namespace)
}

func TestBuildOneAgentAPMImage(t *testing.T) {
	var tag string
	var err error

	tag, err = BuildOneAgentAPMImage("https://test-url.com/api", "default", "all", "")
	assert.NoError(t, err)
	assert.Equal(t, tag, "test-url.com/linux/codemodule")

	tag, err = BuildOneAgentAPMImage("https://test-url.com/api", "default", "dotnet", "")
	assert.NoError(t, err)
	assert.Equal(t, tag, "test-url.com/linux/codemodule:dotnet")

	tag, err = BuildOneAgentAPMImage("https://test-url.com/api", "musl", "java", "")
	assert.NoError(t, err)
	assert.Equal(t, tag, "test-url.com/linux/codemodule-musl:java")

	tag, err = BuildOneAgentAPMImage("https://test-url.com/api", "musl", "php,nginx", "1.123")
	assert.NoError(t, err)
	assert.Equal(t, tag, "test-url.com/linux/codemodule-musl:php-nginx-1.123")

	tag, err = BuildOneAgentAPMImage("https://10.0.0.1/e/abc123456/api", "musl", "all", "1.123")
	assert.NoError(t, err)
	assert.Equal(t, tag, "10.0.0.1/e/abc123456/linux/codemodule-musl:1.123")
}
