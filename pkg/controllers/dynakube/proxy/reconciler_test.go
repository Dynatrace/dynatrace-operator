package proxy

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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

func newTestReconcilerWithInstance(client client.Client) *Reconciler {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:     "https://testing.dev.dynatracelabs.com/api",
			ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
		},
	}

	r := NewReconciler(client, client, scheme.Scheme, instance)

	return r
}

func TestReconcileWithoutProxy(t *testing.T) {
	t.Run(`reconcile dynakube without proxy`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

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

		r := newTestReconcilerWithInstance(testClient)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`ensure no proxy is used when supplying a secret but disabling proxy via feature flag`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakubeName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "https://testing.dev.dynatracelabs.com/api",
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					Value:     "https://proxy:1234",
					ValueFrom: "",
				}}}
		instance.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureActiveGateIgnoreProxy: "true",
			dynatracev1beta1.AnnotationFeatureOneAgentIgnoreProxy:   "true",
		}

		var testClient = fake.NewClientBuilder().WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildSecretName(testDynakubeName),
				Namespace: testNamespace,
			},
		}).Build()

		r := NewReconciler(testClient, testClient, scheme.Scheme, instance)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret

		name := BuildSecretName(testDynakubeName)
		err = testClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestReconcileProxyValue(t *testing.T) {
	t.Run(`reconcile proxy Value - no scheme, no username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl("", "", "", proxyHost, proxyPort)

		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - no scheme, with username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl("", proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - http scheme, no username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyHttpScheme, "", "", proxyHost, proxyPort)

		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - https scheme, no username`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyHttpsScheme, "", "", proxyHost, proxyPort)

		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Empty(t, proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Empty(t, proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpsScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - http scheme`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyHttpScheme, proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy Value - https scheme`, func(t *testing.T) {
		var proxyValue = buildProxyUrl("https", proxyUsername, proxyPassword, proxyHost, proxyPort)

		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpsScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile empty proxy Value`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: ""}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

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
	r := newTestReconcilerWithInstance(testClient)

	t.Run(`reconcile proxy ValueFrom`, func(t *testing.T) {
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{ValueFrom: customProxySecret}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`Change of Proxy ValueFrom to Value`, func(t *testing.T) {
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{ValueFrom: customProxySecret}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret

		r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])

		r.dynakube.Spec.Proxy.ValueFrom = ""
		r.dynakube.Spec.Proxy.Value = buildProxyUrl(proxyHttpScheme, proxyDifferentUsername, proxyPassword, proxyHost, proxyPort)
		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildSecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[passwordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[portField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[hostField])
		assert.Equal(t, []byte(proxyDifferentUsername), proxySecret.Data[usernameField])
		assert.Equal(t, []byte(proxyHttpScheme), proxySecret.Data[schemeField])
	})
	t.Run(`reconcile proxy ValueFrom with non existing secret`, func(t *testing.T) {
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{ValueFrom: "secret"}
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
		Data: map[string][]byte{dynatracev1beta1.ProxyKey: []byte(proxyUrl)},
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
