package secret

import (
	"context"
	"fmt"
	"testing"

	schemeFake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryingest"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testApiToken       = "apiTokenValue"
	testTenantUUID     = "abc12345"
	testKubeSystemUUID = "12345"
)

func TestSecretCreation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates secret if it does not exist", func(t *testing.T) {
		dk := createDynaKube(true)

		testSecret, err := secret.Build(&dk, dk.Name, map[string][]byte{
			dtclient.ApiToken: []byte(testApiToken),
		})
		require.NoError(t, err)

		clt := fake.NewFakeClient(testSecret)

		r := NewReconciler(clt, clt, &dk)

		err = r.ensureTelemetryIngestApiCredentialsSecret(ctx)
		require.NoError(t, err)

		var apiCredsSecret corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: consts.TelemetryApiCredentialsSecretName, Namespace: dk.Namespace}, &apiCredsSecret)
		require.NoError(t, err)
		assert.NotEmpty(t, apiCredsSecret)
		require.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), secretConditionType))
		assert.Equal(t, conditions.SecretCreatedReason, meta.FindStatusCondition(*dk.Conditions(), secretConditionType).Reason)
	})

	t.Run("removes secret if exists but we don't need it", func(t *testing.T) {
		dk := createDynaKube(false)
		conditions.SetSecretCreated(dk.Conditions(), secretConditionType, consts.TelemetryApiCredentialsSecretName)

		objs := []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.TelemetryApiCredentialsSecretName,
					Namespace: dk.Namespace,
				},
			},
		}

		clt := schemeFake.NewClient(objs...)
		r := NewReconciler(clt, clt, &dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var apiTokenSecret corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: consts.TelemetryApiCredentialsSecretName, Namespace: dk.Namespace}, &apiTokenSecret)

		require.Error(t, err)
		assert.Empty(t, apiTokenSecret)
	})
}

func TestEndpoint(t *testing.T) {
	t.Run("in-cluster ActiveGate", func(t *testing.T) {
		dk := createDynaKube(true)
		apiUrl := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", testTenantUUID)
		dk.Spec.APIURL = apiUrl
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{"dynatrace-api"},
		}

		objs := []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.TelemetryApiCredentialsSecretName,
					Namespace: dk.Namespace,
				},
			},
		}

		clt := schemeFake.NewClient(objs...)
		r := NewReconciler(clt, clt, &dk)

		endpoint, err := r.getDtEndpoint()
		require.NoError(t, err)

		expected := fmt.Sprintf("https://%s-activegate.dynatrace.svc/e/%s/api/v2/otlp", dk.Name, testTenantUUID)
		assert.Equal(t, expected, string(endpoint))
	})

	t.Run("public ActiveGate", func(t *testing.T) {
		dk := createDynaKube(true)
		apiUrl := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", testTenantUUID)
		dk.Spec.APIURL = apiUrl

		objs := []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.TelemetryApiCredentialsSecretName,
					Namespace: dk.Namespace,
				},
			},
		}

		clt := schemeFake.NewClient(objs...)
		r := NewReconciler(clt, clt, &dk)

		endpoint, err := r.getDtEndpoint()
		require.NoError(t, err)
		assert.Equal(t, apiUrl+"/v2/otlp", string(endpoint))
	})

	t.Run("managed ActiveGate", func(t *testing.T) {
		dk := createDynaKube(true)
		apiUrl := "https://dynatrace.foobar.com/e/abcdefgh-1234-5678-9abc-deadbeef/api"
		dk.Spec.APIURL = apiUrl

		objs := []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.TelemetryApiCredentialsSecretName,
					Namespace: dk.Namespace,
				},
			},
		}

		clt := schemeFake.NewClient(objs...)
		r := NewReconciler(clt, clt, &dk)

		endpoint, err := r.getDtEndpoint()
		require.NoError(t, err)
		assert.Equal(t, apiUrl+"/v2/otlp", string(endpoint))
	})
}

func createDynaKube(telemetryIngestEnabled bool) dynakube.DynaKube {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
		Spec: dynakube.DynaKubeSpec{},
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

	if telemetryIngestEnabled {
		dk.TelemetryIngest().Spec = &telemetryingest.Spec{}
	} else {
		dk.TelemetryIngest().Spec = nil
	}

	return dk
}
