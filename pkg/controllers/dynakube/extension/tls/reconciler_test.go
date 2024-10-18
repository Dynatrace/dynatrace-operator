package tls

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDynakubeName              = "dynakube"
	testNamespaceName             = "dynatrace"
	testEecPullSecret             = "eec-pull-secret"
	testEecImageRepository        = "repo/dynatrace-eec"
	testEecImageTag               = "1.289.0"
	testTenantUUID                = "abc12345"
	testKubeSystemUUID            = "12345"
	testCustomConfigConfigMapName = "eec-custom-config"
)

var SelfSignedTLSSecretObjectKey = client.ObjectKey{Name: getSelfSignedTLSSecretName(testDynakubeName), Namespace: testNamespaceName}

func TestReconcile(t *testing.T) {
	t.Run("self-signed tls secret is not generated", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "dummy-value"
		fakeClient := fake.NewClient()

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(context.Background(), SelfSignedTLSSecretObjectKey, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		assert.Equal(t, corev1.Secret{}, secret)
	})
	t.Run("self-signed tls secret is generated", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = ""
		fakeClient := fake.NewClient()

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(context.Background(), SelfSignedTLSSecretObjectKey, &secret)

		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})
	t.Run("do not renew self-signed tls secret if it exists", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = ""

		fakeClient := fake.NewClient()
		fakeClient = mockSelfSignedTLSSecret(t, fakeClient, dk)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(context.Background(), SelfSignedTLSSecretObjectKey, &secret)

		require.NoError(t, err)
		require.NotEmpty(t, secret)
	})
	t.Run("self-signed tls secret is deleted", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "dummy-value"

		fakeClient := fake.NewClient()
		fakeClient = mockSelfSignedTLSSecret(t, fakeClient, dk)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(context.Background(), SelfSignedTLSSecretObjectKey, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		assert.Empty(t, secret)
	})
	t.Run("self-signed tls secret is deleted if spec.extensions.enabled is false", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Enabled = false

		fakeClient := fake.NewClient()
		fakeClient = mockSelfSignedTLSSecret(t, fakeClient, dk)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(context.Background(), SelfSignedTLSSecretObjectKey, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		require.Equal(t, corev1.Secret{}, secret)
	})
}

func TestGetTLSSecretName(t *testing.T) {
	t.Run("self-signed tls secret", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = ""

		secretName := GetTLSSecretName(dk)

		assert.Equal(t, getSelfSignedTLSSecretName(dk.Name), secretName)
	})
	t.Run("tlsRefName secret", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "dummy-value"

		secretName := GetTLSSecretName(dk)

		assert.Equal(t, "dummy-value", secretName)
	})
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: dynakube.ExtensionsSpec{
				Enabled: true,
			},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
					ImageRef: image.Ref{
						Repository: testEecImageRepository,
						Tag:        testEecImageTag,
					},
				},
			},
		},

		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testTenantUUID,
				},
				VersionStatus: status.VersionStatus{},
			},
			KubeSystemUUID: testKubeSystemUUID,
		},
	}
}

func mockSelfSignedTLSSecret(t *testing.T, client client.Client, dk *dynakube.DynaKube) client.Client {
	tlsSecret := getSelfSignedTLSSecret(dk)

	err := client.Create(context.Background(), &tlsSecret)
	require.NoError(t, err)

	return client
}

func getSelfSignedTLSSecret(dk *dynakube.DynaKube) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSelfSignedTLSSecretName(dk.Name),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			consts.TLSCrtDataName: []byte("super-cert"),
			consts.TLSKeyDataName: []byte("super-key"),
		},
	}
}
