package tls

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/stretchr/testify/require"
	"k8s.io/api/apps/v1"
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

var TLSSecretObjectKey = client.ObjectKey{Name: getSelfSignedTLSSecretName(testDynakubeName), Namespace: testNamespaceName}

func TestReconcile(t *testing.T) {
	t.Run("self-signed tls secret is not generated", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "dummy-value"
		fakeClient := fake.NewClient()

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		reconciler.Reconcile(context.Background())

		var secret corev1.Secret
		err := fakeClient.Get(context.Background(), TLSSecretObjectKey, &secret)

		require.True(t, k8serrors.IsNotFound(err))
		require.Equal(t, corev1.Secret{}, secret)
	})
	t.Run("self-signed tls secret is generated", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = ""
		fakeClient := fake.NewClient()

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		reconciler.Reconcile(context.Background())

		var secret corev1.Secret
		err := fakeClient.Get(context.Background(), TLSSecretObjectKey, &secret)

		require.NoError(t, err)
		require.NotNil(t, secret)
	})
	t.Run("do not renew self-signed tls secret if it exists", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = ""

		expectedTLSSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getSelfSignedTLSSecretName(dk.Name),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{
				consts.TLSCrtDataName: []byte("super-cert"),
				consts.TLSKeyDataName: []byte("super-key"),
			},
		}

		fakeClient := fake.NewClient()
		fakeClient.Create(context.Background(), &expectedTLSSecret)

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		reconciler.Reconcile(context.Background())

		var secret corev1.Secret
		err := fakeClient.Get(context.Background(), TLSSecretObjectKey, &secret)

		require.NoError(t, err)
		require.Equal(t, expectedTLSSecret, secret)
	})
	t.Run("update eec and otelc statefulsets", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient()
		fakeClient.Create(context.Background(), &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{}})

		reconciler := NewReconciler(fakeClient, fakeClient, dk)

		reconciler.Reconcile(context.Background())

		var secret corev1.Secret
		err := fakeClient.Get(context.Background(), TLSSecretObjectKey, &secret)

		require.NoError(t, err)
		require.NotNil(t, secret)
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
					ImageRef: dynakube.ImageRefSpec{
						Repository: testEecImageRepository,
						Tag:        testEecImageTag,
					},
				},
			},
		},

		Status: dynakube.DynaKubeStatus{
			ActiveGate: dynakube.ActiveGateStatus{
				ConnectionInfoStatus: dynakube.ActiveGateConnectionInfoStatus{
					ConnectionInfoStatus: dynakube.ConnectionInfoStatus{
						TenantUUID: testTenantUUID,
					},
				},
				VersionStatus: status.VersionStatus{},
			},
			KubeSystemUUID: testKubeSystemUUID,
		},
	}
}
