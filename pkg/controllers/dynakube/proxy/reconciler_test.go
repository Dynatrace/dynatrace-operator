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
		err = r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`ensure proxy secret deleted`, func(t *testing.T) {
		var testClient = fake.NewClientBuilder().WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildProxySecretName(testDynakubeName),
				Namespace: testNamespace,
			},
		}).Build()
		r := newTestReconcilerWithInstance(testClient)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		require.Error(t, err)
		assert.Empty(t, proxySecret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestReconcileProxyValue(t *testing.T) {
	t.Run(`reconcile proxy Value`, func(t *testing.T) {
		var proxyValue = buildProxyUrl(proxyUsername, proxyPassword, proxyHost, proxyPort)
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: proxyValue}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[proxyPasswordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[proxyPortField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[proxyHostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[proxyUsernameField])
	})
	t.Run(`reconcile empty proxy Value`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: ""}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(nil), proxySecret.Data[proxyPasswordField])
		assert.Equal(t, []byte(nil), proxySecret.Data[proxyPortField])
		assert.Equal(t, []byte(nil), proxySecret.Data[proxyHostField])
		assert.Equal(t, []byte(nil), proxySecret.Data[proxyUsernameField])
	})
}

func TestReconcileProxyValueFrom(t *testing.T) {
	var proxyUrl = buildProxyUrl(proxyUsername, proxyPassword, proxyHost, proxyPort)
	var testClient = fake.NewClientBuilder().WithObjects(createProxySecret(proxyUrl)).Build()
	r := newTestReconcilerWithInstance(testClient)

	t.Run(`reconcile proxy ValueFrom`, func(t *testing.T) {
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{ValueFrom: customProxySecret}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[proxyPasswordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[proxyPortField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[proxyHostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[proxyUsernameField])
	})
	t.Run(`Change of Proxy ValueFrom to Value`, func(t *testing.T) {
		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{ValueFrom: customProxySecret}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var proxySecret corev1.Secret
		r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[proxyPasswordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[proxyPortField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[proxyHostField])
		assert.Equal(t, []byte(proxyUsername), proxySecret.Data[proxyUsernameField])

		r.dynakube.Spec.Proxy.ValueFrom = ""
		r.dynakube.Spec.Proxy.Value = buildProxyUrl(proxyDifferentUsername, proxyPassword, proxyHost, proxyPort)
		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		_ = r.client.Get(context.Background(), client.ObjectKey{Name: BuildProxySecretName(testDynakubeName), Namespace: testNamespace}, &proxySecret)

		assert.Equal(t, []byte(proxyPassword), proxySecret.Data[proxyPasswordField])
		assert.Equal(t, []byte(proxyPort), proxySecret.Data[proxyPortField])
		assert.Equal(t, []byte(proxyHost), proxySecret.Data[proxyHostField])
		assert.Equal(t, []byte(proxyDifferentUsername), proxySecret.Data[proxyUsernameField])
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

func buildProxyUrl(username string, password string, host string, port string) string {
	return "http://" + username + ":" + password + "@" + host + ":" + port
}
