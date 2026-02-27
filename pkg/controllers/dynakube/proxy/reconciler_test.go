package proxy

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	dtfake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakubeName       = "test-dynakube"
	testNamespace          = "test-namespace"
	customProxySecret      = "testProxy"
	proxyUsername          = "testUser"
	proxyPassword          = "secretValue"
	proxyPort              = "1020"
	proxyHost              = "proxyserver.net"
	proxyHTTPScheme        = "http"
	proxyHTTPSScheme       = "https"
	proxyDifferentUsername = "differentUsername"
)

func createK8sClientWithProxySecret(t *testing.T) client.Client {
	t.Helper()
	mockK8sClient := dtfake.NewClient()
	_ = mockK8sClient.Create(t.Context(),
		&corev1.Secret{
			Data: map[string][]byte{connectioninfo.TenantTokenKey: []byte("test-token")},
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildSecretName(testDynakubeName),
				Namespace: testNamespace,
			},
		},
	)

	return mockK8sClient
}

func createDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:     "https://testing.dev.dynatracelabs.com/api",
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
		},
	}
}

func createDynaKubeWithProxy(proxy *value.Source) *dynakube.DynaKube {
	dk := createDynaKube()
	dk.Spec.Proxy = proxy

	return dk
}

func TestReconcileWithoutProxy(t *testing.T) {
	t.Run("reconcile dynakube without proxy", func(t *testing.T) {
		testClient := fake.NewFakeClient()
		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKube())

		require.NoError(t, err)

		var proxySecret corev1.Secret

		err = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)
		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("ensure proxy secret deleted", func(t *testing.T) {
		var testClient = fake.NewClientBuilder().WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildSecretName(testDynakubeName),
				Namespace: testNamespace,
			},
		}).Build()

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKube())

		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("ensure no proxy is used when supplying a secret but disabling proxy via feature flag", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakubeName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://testing.dev.dynatracelabs.com/api",
				Proxy: &value.Source{
					Value:     "https://proxy:1234",
					ValueFrom: "",
				}}}
		dk.Annotations = map[string]string{
			exp.AGIgnoreProxyKey:  "true", //nolint:staticcheck
			exp.OAProxyIgnoredKey: "true", //nolint:staticcheck
		}

		var testClient = createK8sClientWithProxySecret(t)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), dk)

		require.NoError(t, err)

		var proxySecret corev1.Secret

		name := BuildSecretName(testDynakubeName)
		err = r.client.Get(t.Context(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestReconcileProxyValue(t *testing.T) {
	testClient := fake.NewFakeClient()

	t.Run("reconcile proxy Value - no scheme, no username", func(t *testing.T) {
		var proxyValue = buildProxyURL("", "", "", proxyHost, proxyPort)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile proxy Value - no scheme, with username", func(t *testing.T) {
		var proxyValue = buildProxyURL("", proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile proxy Value - http scheme, no username", func(t *testing.T) {
		var proxyValue = buildProxyURL(proxyHTTPScheme, "", "", proxyHost, proxyPort)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile proxy Value - https scheme, no username", func(t *testing.T) {
		var proxyValue = buildProxyURL(proxyHTTPSScheme, "", "", proxyHost, proxyPort)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPSScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile proxy Value - http scheme", func(t *testing.T) {
		var proxyValue = buildProxyURL(proxyHTTPScheme, proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile proxy Value - https scheme", func(t *testing.T) {
		var proxyValue = buildProxyURL("https", proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPSScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile empty proxy Value", func(t *testing.T) {
		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{Value: ""}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(nil), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(nil), proxySecret.Data[portField])
		assert.Equal(t, []byte(nil), proxySecret.Data[hostField])
		assert.Equal(t, []byte(nil), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(nil), proxySecret.Data[schemeField])
	})
}

func TestReconcileProxyValueFrom(t *testing.T) {
	var proxyURL = buildProxyURL(proxyHTTPScheme, proxyUsername, proxyPassword, proxyHost, proxyPort)

	var testClient = fake.NewClientBuilder().WithObjects(createProxySecret(proxyURL)).Build()

	t.Run("reconcile proxy ValueFrom", func(t *testing.T) {
		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{ValueFrom: customProxySecret}))
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])
	})
	t.Run("Change of Proxy ValueFrom to Value", func(t *testing.T) {
		dk := createDynaKubeWithProxy(&value.Source{ValueFrom: customProxySecret})
		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), dk)
		require.NoError(t, err)

		var proxySecret corev1.Secret

		r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])

		dk.Spec.Proxy.ValueFrom = ""
		dk.Spec.Proxy.Value = buildProxyURL(proxyHTTPScheme, proxyDifferentUsername, proxyPassword, proxyHost, proxyPort)
		err = r.Reconcile(t.Context(), dk)

		require.NoError(t, err)

		_ = r.client.Get(t.Context(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyDifferentUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHTTPScheme), proxySecret.Data[schemeField])
	})
	t.Run("reconcile proxy ValueFrom with non existing secret", func(t *testing.T) {
		r := NewReconciler(testClient, testClient)
		err := r.Reconcile(t.Context(), createDynaKubeWithProxy(&value.Source{ValueFrom: "secret"}))

		require.Error(t, err)
	})
}

func createProxySecret(proxyURL string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customProxySecret,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{dynakube.ProxyKey: []byte(proxyURL)},
	}
}

func buildProxyURL(scheme string, username string, password string, host string, port string) string {
	url := ""
	if scheme != "" {
		url = scheme + "://"
	}

	if username != "" {
		url = url + username + ":" + password + "@"
	}

	return url + host + ":" + port
}
