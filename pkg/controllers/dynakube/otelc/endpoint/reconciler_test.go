package endpoint

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	schemeFake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
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
	testAPIToken       = "apiTokenValue"
	testTenantUUID     = "abc12345"
	testKubeSystemUUID = "12345"
)

func TestConfigMapCreation(t *testing.T) {
	ctx := t.Context()

	t.Run("creates config map if it does not exist", func(t *testing.T) {
		dk := createDynaKube(true)

		testConfigMap, err := k8sconfigmap.Build(&dk, dk.Name, map[string]string{
			dtclient.APIToken: testAPIToken,
		})
		require.NoError(t, err)

		clt := fake.NewFakeClient(testConfigMap)

		r := NewReconciler(clt, clt, &dk)

		err = r.reconcileConfigMap(ctx)
		require.NoError(t, err)

		var apiEndpointConfigMap corev1.ConfigMap
		err = clt.Get(ctx, types.NamespacedName{Name: consts.OtlpAPIEndpointConfigMapName, Namespace: dk.Namespace}, &apiEndpointConfigMap)
		require.NoError(t, err)
		assert.NotEmpty(t, apiEndpointConfigMap)
		require.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), configMapConditionType))
		assert.Equal(t, conditions.ConfigMapCreatedOrUpdatedReason, meta.FindStatusCondition(*dk.Conditions(), configMapConditionType).Reason)
	})

	t.Run("removes config map if exists but we don't need it", func(t *testing.T) {
		dk := createDynaKube(false)
		conditions.SetConfigMapCreatedOrUpdated(dk.Conditions(), configMapConditionType, consts.OtlpAPIEndpointConfigMapName)

		objs := []client.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.OtlpAPIEndpointConfigMapName,
					Namespace: dk.Namespace,
				},
			},
		}

		clt := schemeFake.NewClient(objs...)
		r := NewReconciler(clt, clt, &dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var apiEndpointConfigmap corev1.ConfigMap
		err = clt.Get(ctx, types.NamespacedName{Name: consts.OtlpAPIEndpointConfigMapName, Namespace: dk.Namespace}, &apiEndpointConfigmap)

		require.Error(t, err)
		assert.Empty(t, apiEndpointConfigmap)
	})
}

func Test_generateData(t *testing.T) {
	tests := []struct {
		name         string
		apiURL       string
		expectedData map[string]string
		inClusterAg  bool
	}{
		{
			name:         "in-cluster ActiveGate",
			apiURL:       fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", testTenantUUID),
			inClusterAg:  true,
			expectedData: map[string]string{"DT_ENDPOINT": fmt.Sprintf("https://test-dk-activegate.test-namespace/e/%s/api/v2/otlp", testTenantUUID)},
		},
		{
			name:         "public ActiveGate",
			apiURL:       fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", testTenantUUID),
			inClusterAg:  false,
			expectedData: map[string]string{"DT_ENDPOINT": fmt.Sprintf("https://%s.dev.dynatracelabs.com/api/v2/otlp", testTenantUUID)},
		},
		{
			name:         "managed ActiveGate",
			apiURL:       "https://dynatrace.foobar.com/e/abcdefgh-1234-5678-9abc-deadbeef/api",
			inClusterAg:  false,
			expectedData: map[string]string{"DT_ENDPOINT": "https://dynatrace.foobar.com/e/abcdefgh-1234-5678-9abc-deadbeef/api/v2/otlp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dk := createDynaKube(true)
			dk.Spec.APIURL = tt.apiURL

			if tt.inClusterAg {
				dk.Spec.ActiveGate = activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{"dynatrace-api"},
				}
			}

			objs := []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      consts.OtlpAPIEndpointConfigMapName,
						Namespace: dk.Namespace,
					},
				},
			}

			clt := schemeFake.NewClient(objs...)
			r := NewReconciler(clt, clt, &dk)

			data, err := r.generateData()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedData, data)
		})
	}
}

func createDynaKube(telemetryIngestEnabled bool) dynakube.DynaKube {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-namespace",
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
