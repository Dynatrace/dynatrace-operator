package configsecret

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	dkName      = "test-name"
	dkNamespace = "test-namespace"
	tokenValue  = "test-token"
)

func TestReconcile(t *testing.T) {
	ctx := t.Context()

	t.Run("Only clean up if not standalone", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		conditions.SetSecretCreated(dk.Conditions(), LmcConditionType, "testing")

		mockK8sClient := createK8sClientWithConfigSecret(t)

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		var secret corev1.Secret
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: GetSecretName((dk.Name)), Namespace: dk.Namespace}, &secret)
		require.True(t, k8serrors.IsNotFound(err))

		condition := meta.FindStatusCondition(*dk.Conditions(), LmcConditionType)
		require.Nil(t, condition)
	})

	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dk := createDynakube(true)

		mockK8sClient := createK8sClientWithOneAgentTenantSecret(t, dk, tokenValue)

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		checkSecretForValue(t, mockK8sClient, dk)

		condition := meta.FindStatusCondition(*dk.Conditions(), LmcConditionType)
		require.NotNil(t, condition)
		oldTransitionTime := condition.LastTransitionTime
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		err = reconciler.Reconcile(t.Context())

		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, dk)
	})

	t.Run("Create and update works with no-proxy/proxy/network-zone", func(t *testing.T) {
		dk := createDynakube(true)

		mockK8sClient := createK8sClientWithOneAgentTenantSecret(t, dk, tokenValue)

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		checkSecretForValue(t, mockK8sClient, dk)

		condition := meta.FindStatusCondition(*dk.Conditions(), LmcConditionType)
		require.NotNil(t, condition)
		oldTransitionTime := condition.LastTransitionTime
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		reconciler.dk.Spec.NetworkZone = "test-zone"
		reconciler.dk.Spec.Proxy = &value.Source{
			Value: "test-proxy",
		}
		reconciler.dk.Annotations = map[string]string{
			exp.NoProxyKey: "test-no-proxy",
		}

		err = reconciler.Reconcile(t.Context())

		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, dk)
	})
	t.Run("Only runs when required, and cleans up condition + secret", func(t *testing.T) {
		dk := createDynakube(false)

		mockK8sClient := createK8sClientWithOneAgentTenantSecret(t, dk, tokenValue)
		conditions.SetSecretCreated(dk.Conditions(), LmcConditionType, "this is a test")

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, *dk.Conditions())

		var secretConfig corev1.Secret
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      GetSecretName(dk.Name),
			Namespace: dk.Namespace,
		}, &secretConfig)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("problem with k8s request => visible in conditions", func(t *testing.T) {
		dk := createDynakube(true)

		boomClient := createBOOMK8sClient(t)

		reconciler := NewReconciler(boomClient,
			boomClient, dk)

		err := reconciler.Reconcile(t.Context())

		require.Error(t, err)
		require.Len(t, *dk.Conditions(), 1)
		condition := meta.FindStatusCondition(*dk.Conditions(), LmcConditionType)
		assert.Equal(t, conditions.KubeAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func checkSecretForValue(t *testing.T, k8sClient client.Client, dk *dynakube.DynaKube) {
	t.Helper()

	var secret corev1.Secret
	err := k8sClient.Get(t.Context(), client.ObjectKey{Name: GetSecretName(dk.Name), Namespace: dk.Namespace}, &secret)
	require.NoError(t, err)

	deploymentConfig, ok := secret.Data[DeploymentConfigFilename]
	require.True(t, ok)

	tenantUUID, err := dk.TenantUUID()
	require.NoError(t, err)

	expectedLines := []string{
		serverKey + "=" + fmt.Sprintf("{%s}", dk.Status.OneAgent.ConnectionInfoStatus.Endpoints),
		tenantKey + "=" + tenantUUID,
		tenantTokenKey + "=" + tokenValue,
		hostIDSourceKey + "=k8s-node-name",
	}

	if dk.Spec.NetworkZone != "" {
		expectedLines = append(expectedLines, networkZoneKey+"="+dk.Spec.NetworkZone)
	}

	if dk.HasProxy() {
		proxyURL, err := dk.Proxy(t.Context(), k8sClient)
		require.NoError(t, err)
		expectedLines = append(expectedLines, proxyKey+"="+proxyURL)
	}

	if createNoProxyValue(*dk) != "" {
		expectedLines = append(expectedLines, noProxyKey+"="+createNoProxyValue(*dk))
	}

	split := strings.Split(strings.Trim(string(deploymentConfig), "\n"), "\n")
	require.Len(t, split, len(expectedLines))

	for _, line := range split {
		assert.Contains(t, expectedLines, line)
	}
}

func createDynakube(isLogMonitoringEnabled bool) *dynakube.DynaKube {
	var logMonitoringSpec *logmonitoring.Spec
	if isLogMonitoringEnabled {
		logMonitoringSpec = &logmonitoring.Spec{}
	}

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dkNamespace,
			Name:      dkName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:        "test-url",
			LogMonitoring: logMonitoringSpec,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: "test-uuid",
						Endpoints:  "https://endpoint1.com;https://endpoint2.com",
					},
				},
			},
		},
	}
}

func createBOOMK8sClient(t *testing.T) client.Client {
	t.Helper()

	boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			return errors.New("BOOM")
		},
		Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			return errors.New("BOOM")
		},
		Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			return errors.New("BOOM")
		},
		Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}

func createK8sClientWithOneAgentTenantSecret(t *testing.T, dk *dynakube.DynaKube, token string) client.Client {
	t.Helper()

	mockK8sClient := fake.NewClient()
	_ = mockK8sClient.Create(t.Context(),
		&corev1.Secret{
			Data: map[string][]byte{connectioninfo.TenantTokenKey: []byte(token)},
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.OneAgent().GetTenantSecret(),
				Namespace: dkNamespace,
			},
		},
	)

	return mockK8sClient
}

func createK8sClientWithConfigSecret(t *testing.T) client.Client {
	mockK8sClient := fake.NewClient()
	_ = mockK8sClient.Create(t.Context(),
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetSecretName(dkName),
				Namespace: dkNamespace,
			},
		},
	)

	return mockK8sClient
}

func TestAddAnnotations(t *testing.T) {
	type testCase struct {
		title       string
		annotations map[string]string
		dk          dynakube.DynaKube
		expectedOut map[string]string
	}

	cases := []testCase{
		{
			title: "nil map doesn't break it",
			dk: dynakube.DynaKube{
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
							ConnectionInfo: communication.ConnectionInfo{
								TenantTokenHash: "hash",
							},
						},
					},
				},
			},
			annotations: nil,
			expectedOut: map[string]string{
				TokenHashAnnotationKey: "hash",
			},
		},
		{
			title: "existing annotations are untouched",
			dk: dynakube.DynaKube{
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
							ConnectionInfo: communication.ConnectionInfo{
								TenantTokenHash: "hash",
							},
						},
					},
				},
			},
			annotations: map[string]string{
				"other": "annotation",
			},
			expectedOut: map[string]string{
				"other":                "annotation",
				TokenHashAnnotationKey: "hash",
			},
		},
		{
			title: "network-zone respected",
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					NetworkZone: "test-zone",
				},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
							ConnectionInfo: communication.ConnectionInfo{
								TenantTokenHash: "hash",
							},
						},
					},
				},
			},
			annotations: map[string]string{},
			expectedOut: map[string]string{
				TokenHashAnnotationKey:   "hash",
				NetworkZoneAnnotationKey: "test-zone",
			},
		},
		{
			title: "proxy respected",
			dk: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					Proxy: &value.Source{Value: "doesn't matter"},
				},
				Status: dynakube.DynaKubeStatus{
					ProxyURLHash: "proxy-hash",
					OneAgent: oneagent.Status{
						ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
							ConnectionInfo: communication.ConnectionInfo{
								TenantTokenHash: "hash",
							},
						},
					},
				},
			},
			annotations: map[string]string{},
			expectedOut: map[string]string{
				TokenHashAnnotationKey: "hash",
				ProxyHashAnnotationKey: "proxy-hash",
			},
		},
		{
			title: "no-proxy respected",
			dk: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						exp.NoProxyKey: "no-proxy",
					},
				},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
							ConnectionInfo: communication.ConnectionInfo{
								TenantTokenHash: "hash",
							},
						},
					},
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.1.1.1", "2.2.2.2"},
					},
				},
			},
			annotations: map[string]string{},
			expectedOut: map[string]string{
				TokenHashAnnotationKey: "hash",
				NoProxyAnnotationKey:   "no-proxy,1.1.1.1,2.2.2.2",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			out := AddAnnotations(c.annotations, c.dk)

			require.NotEqual(t, c.annotations, out)
			assert.Equal(t, c.expectedOut, out)
		})
	}
}
