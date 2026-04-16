package dynatraceclient

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName        = "test-name"
	testNetworkZone = "zone-1"
)

func TestOptions(t *testing.T) {
	createTestDynakubeWithProxy := func(t *testing.T, proxy value.Source) *dynakube.DynaKube {
		t.Helper()

		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Proxy: &proxy,
			},
		}
		dk.Namespace = testNamespace

		return dk
	}

	t.Run("Test append network zone", func(t *testing.T) {
		opts := newOptions(t.Context())

		assert.NotNil(t, opts)
		assert.Empty(t, opts.Opts)

		opts.appendNetworkZone("")

		assert.Empty(t, opts.Opts)

		opts.appendNetworkZone(testNetworkZone)

		assert.NotEmpty(t, opts.Opts)
	})
	t.Run("Test append cert check", func(t *testing.T) {
		opts := newOptions(t.Context())

		assert.NotNil(t, opts)
		assert.Empty(t, opts.Opts)

		opts.appendCertCheck(false)

		assert.NotNil(t, opts)
		assert.NotEmpty(t, opts.Opts)

		opts = newOptions(t.Context())
		opts.appendCertCheck(true)

		assert.NotNil(t, opts)
		assert.NotEmpty(t, opts.Opts)
	})
	t.Run("Test append proxy settings", func(t *testing.T) {
		opts := newOptions(t.Context())

		assert.NotNil(t, opts)
		assert.Empty(t, opts.Opts)

		err := opts.appendProxySettings(nil, nil)
		require.NoError(t, err)
		assert.Empty(t, opts.Opts)

		err = opts.appendProxySettings(nil, createTestDynakubeWithProxy(t, value.Source{Value: testValue}))

		require.NoError(t, err)
		assert.NotEmpty(t, opts.Opts)

		fakeClient := fake.NewClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					dynakube.ProxyKey: []byte(testValue),
				},
			})
		opts = newOptions(t.Context())
		err = opts.appendProxySettings(fakeClient, createTestDynakubeWithProxy(t, value.Source{ValueFrom: testName}))

		require.NoError(t, err)
		assert.NotEmpty(t, opts.Opts)
	})
	t.Run("AppendProxySettings handles missing or malformed secret", func(t *testing.T) {
		fakeClient := fake.NewClient()
		opts := newOptions(t.Context())
		err := opts.appendProxySettings(fakeClient, createTestDynakubeWithProxy(t, value.Source{ValueFrom: testName}))

		require.Error(t, err)
		assert.Empty(t, opts.Opts)

		fakeClient = fake.NewClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{},
			})
		opts = newOptions(t.Context())
		err = opts.appendProxySettings(fakeClient, createTestDynakubeWithProxy(t, value.Source{ValueFrom: testName}))

		require.Error(t, err)
		assert.Empty(t, opts.Opts)
	})
	t.Run("Test append trusted certificates", func(t *testing.T) {
		opts := newOptions(t.Context())

		assert.NotNil(t, opts)
		assert.Empty(t, opts.Opts)

		err := opts.appendTrustedCerts(nil, "", "")

		require.NoError(t, err)
		assert.Empty(t, opts.Opts)

		fakeClient := fake.NewClient(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					dynakube.TrustedCAKey: testValue,
				}})
		err = opts.appendTrustedCerts(fakeClient, testName, testNamespace)

		require.NoError(t, err)
		assert.NotEmpty(t, opts.Opts)
	})
	t.Run("AppendTrustedCerts handles missing or malformed config map", func(t *testing.T) {
		opts := newOptions(t.Context())

		assert.NotNil(t, opts)
		assert.Empty(t, opts.Opts)

		fakeClient := fake.NewClient()
		err := opts.appendTrustedCerts(fakeClient, testName, testNamespace)

		require.Error(t, err)
		assert.Empty(t, opts.Opts)

		fakeClient = fake.NewClient(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string]string{}})
		err = opts.appendTrustedCerts(fakeClient, testName, testNamespace)

		require.Error(t, err)
		assert.Empty(t, opts.Opts)
	})
}
