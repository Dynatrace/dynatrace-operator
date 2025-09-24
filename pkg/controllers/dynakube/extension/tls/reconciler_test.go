package tls

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
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

func TestReconcile(t *testing.T) {
	t.Run("self-signed tls secret is not generated", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "dummy-value"
		fakeClient := fake.NewClient()

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret

		key := client.ObjectKey{Name: dk.Extensions().GetSelfSignedTLSSecretName(), Namespace: testNamespaceName}
		err = fakeClient.Get(context.Background(), key, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		assert.Equal(t, corev1.Secret{}, secret)
		assert.Empty(t, dk.Conditions())
	})
	t.Run("self-signed tls secret is generated", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = ""
		fakeClient := fake.NewClient()

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret

		key := client.ObjectKey{Name: dk.Extensions().GetSelfSignedTLSSecretName(), Namespace: testNamespaceName}
		err = fakeClient.Get(context.Background(), key, &secret)

		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		require.NotEmpty(t, dk.Conditions())
		assert.Equal(t, conditionType, (*dk.Conditions())[0].Type)
		assert.Equal(t, metav1.ConditionTrue, (*dk.Conditions())[0].Status)
		assert.Equal(t, conditions.SecretCreatedReason, (*dk.Conditions())[0].Reason)
		assert.Equal(t, "dynakube-extensions-controller-tls created", (*dk.Conditions())[0].Message)
	})
	t.Run("do not renew self-signed tls secret if it exists", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = ""
		conditions.SetSecretCreated(dk.Conditions(), conditionType, "dynakube-extensions-controller-tls")

		fakeClient := fake.NewClient()
		fakeClient = mockSelfSignedTLSSecret(t, fakeClient, dk)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret

		key := client.ObjectKey{Name: dk.Extensions().GetSelfSignedTLSSecretName(), Namespace: testNamespaceName}
		err = fakeClient.Get(context.Background(), key, &secret)

		require.NoError(t, err)
		require.NotEmpty(t, secret)
		assert.NotEmpty(t, dk.Conditions())
	})
	t.Run("self-signed tls secret is deleted", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "dummy-value"
		conditions.SetSecretCreated(dk.Conditions(), conditionType, "dynakube-extensions-controller-tls")

		fakeClient := fake.NewClient()
		fakeClient = mockSelfSignedTLSSecret(t, fakeClient, dk)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret

		key := client.ObjectKey{Name: dk.Extensions().GetSelfSignedTLSSecretName(), Namespace: testNamespaceName}
		err = fakeClient.Get(context.Background(), key, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		assert.Empty(t, secret)
		assert.Empty(t, dk.Conditions())
	})
	t.Run("self-signed tls secret is deleted if spec.extensions.enabled is false", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions = nil
		conditions.SetSecretCreated(dk.Conditions(), conditionType, "dynakube-extensions-controller-tls")

		fakeClient := fake.NewClient()
		fakeClient = mockSelfSignedTLSSecret(t, fakeClient, dk)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret

		key := client.ObjectKey{Name: dk.Extensions().GetSelfSignedTLSSecretName(), Namespace: testNamespaceName}
		err = fakeClient.Get(context.Background(), key, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		assert.Equal(t, corev1.Secret{}, secret)
		assert.Empty(t, dk.Conditions())
	})
}

func TestGetTLSSecretName(t *testing.T) {
	t.Run("self-signed tls secret", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = ""

		secretName := dk.Extensions().GetTLSSecretName()
		assert.Equal(t, dk.Extensions().GetSelfSignedTLSSecretName(), secretName)
	})
	t.Run("tlsRefName secret", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "dummy-value"

		secretName := dk.Extensions().GetTLSSecretName()
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
			Extensions: &extensions.Spec{PrometheusSpec: &extensions.PrometheusSpec{}, Databases: []extensions.Database{}},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
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
			Name:      dk.Extensions().GetTLSSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			consts.TLSCrtDataName: []byte("super-cert"),
			consts.TLSKeyDataName: []byte("super-key"),
		},
	}
}
