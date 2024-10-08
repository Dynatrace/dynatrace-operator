package proxy

import (
	"context"
	"testing"

	dtfake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
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
	proxyHttpScheme        = "http"
	proxyHttpsScheme       = "https"
	proxyDifferentUsername = "differentUsername"
)

func createK8sClientWithProxySecret() client.Client {
	mockK8sClient := dtfake.NewClient()
	_ = mockK8sClient.Create(context.Background(),
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

func newTestReconciler(client client.Client) controllers.Reconciler {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:     "https://testing.dev.dynatracelabs.com/api",
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
		},
	}

	r := NewReconciler(client, client, dk)

	return r
}

func newTestReconcilerWithCustomDynaKube(client client.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	r := NewReconciler(client, client, dk)

	return r
}

func createDynaKubeWithProxy(proxy *value.Source) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:     "https://testing.dev.dynatracelabs.com/api",
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
			Proxy:      proxy,
		},
	}
}

func TestReconcileWithoutProxy(t *testing.T) {
	t.Run(`reconcile dynakube without proxy`, func(t *testing.T) {
		testClient := fake.NewFakeClient()
		r := newTestReconciler(testClient)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret

		err = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)
		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`ensure proxy secret deleted`, func(t *testing.T) {
		var testClient = fake.NewClientBuilder().WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildSecretName(testDynakubeName),
				Namespace: testNamespace,
			},
		}).Build()

		r := newTestReconciler(testClient)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`ensure no proxy is used when supplying a secret but disabling proxy via feature flag`, func(t *testing.T) {
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
			dynakube.AnnotationFeatureActiveGateIgnoreProxy: "true", //nolint:staticcheck
			dynakube.AnnotationFeatureOneAgentIgnoreProxy:   "true", //nolint:staticcheck
		}

		var testClient = createK8sClientWithProxySecret()

		r := NewReconciler(testClient, testClient, dk)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret

		name := BuildSecretName(testDynakubeName)
		err = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestReconcileProxyValue(t *testing.T) {
	testClient := fake.NewFakeClient()

	t.Run(`reconcile proxy Value - no scheme, no username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl("", "", "", proxyHost, proxyPort)

		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - no scheme, with username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl("", proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - http scheme, no username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyHttpScheme, "", "", proxyHost, proxyPort)

		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - https scheme, no username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyHttpsScheme, "", "", proxyHost, proxyPort)

		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpsScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - http scheme`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyHttpScheme, proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - https scheme`, func(t *testing.T) {
		var proxyValue = buildProxyUrl("https", proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: proxyValue}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpsScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile empty proxy Value`, func(t *testing.T) {
		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{Value: ""}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(nil), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(nil), proxySecret.Data[portField])
		assert.Equal(t, []byte(nil), proxySecret.Data[hostField])
		assert.Equal(t, []byte(nil), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(nil), proxySecret.Data[schemeField])
	})
}

func TestReconcileProxyValueFrom(t *testing.T) {
	var proxyUrl = buildProxyUrl(proxyHttpScheme, proxyUsername, proxyPassword, proxyHost, proxyPort)

	var testClient = fake.NewClientBuilder().WithObjects(createProxySecret(proxyUrl)).Build()

	t.Run(`reconcile proxy ValueFrom`, func(t *testing.T) {
		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{ValueFrom: customProxySecret}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`Change of Proxy ValueFrom to Value`, func(t *testing.T) {
		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{ValueFrom: customProxySecret}))
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret

		r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])

		r.(*Reconciler).dk.Spec.Proxy.ValueFrom = ""
		r.(*Reconciler).dk.Spec.Proxy.Value = buildProxyUrl(proxyHttpScheme, proxyDifferentUsername, proxyPassword, proxyHost, proxyPort)
		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		_ = r.(*Reconciler).client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyDifferentUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy ValueFrom with non existing secret`, func(t *testing.T) {
		r := newTestReconcilerWithCustomDynaKube(testClient, createDynaKubeWithProxy(&value.Source{ValueFrom: "secret"}))
		err := r.Reconcile(context.Background())

		require.Error(t, err)
	})
}

func createProxySecret(proxyUrl string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customProxySecret,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{dynakube.ProxyKey: []byte(proxyUrl)},
	}
}

func buildProxyUrl(scheme string, username string, password string, host string, port string) string {
	url := ""
	if scheme != "" {
		url = scheme + "://"
	}

	if username != "" {
		url = url + username + ":" + password + "@"
	}

	return url + host + ":" + port
}
