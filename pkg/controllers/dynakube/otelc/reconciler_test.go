package otelc

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/endpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testToken          = "apiTokenValue"
	testTenantUUID     = "abc12345"
	testKubeSystemUUID = "12345"
	testAPIURL         = "test-apiurl"
)

func TestNoProxyConsistency(t *testing.T) {
	ctx := t.Context()

	t.Run("NO_PROXY matches DT_ENDPOINT if proxy is set and local AG defined", func(t *testing.T) {
		dk := createDynaKube(true)

		clt := createClient(t, &dk)

		dtEndpoint, noProxy := reconcile(t, ctx, clt, dk)
		// NO_PROXY    = noProxy
		// DT_ENDPOINT = scheme :// hostname=noProxy / path
		assert.Contains(t, dtEndpoint, "/"+noProxy+"/")
	})

	t.Run("NO_PROXY matches DT_ENDPOINT if proxy is set and cluster AG defined", func(t *testing.T) {
		dk := createDynaKube(false)

		clt := createClient(t, &dk)

		dtEndpoint, noProxy := reconcile(t, ctx, clt, dk)
		assert.Equal(t, dk.APIURL()+"/v2/otlp", dtEndpoint)
		assert.Empty(t, noProxy)
	})
}

func createClient(t *testing.T, dk *dynakube.DynaKube) client.WithWatch {
	testTokensSecret, err := secret.Build(dk, dk.Name, map[string][]byte{
		dtclient.APIToken:        []byte(testToken),
		dtclient.DataIngestToken: []byte(testToken),
	})
	require.NoError(t, err)

	testConfig, err := configmap.Build(dk, dk.Name+consts.TelemetryCollectorConfigmapSuffix, map[string]string{consts.ConfigFieldName: "test"})
	require.NoError(t, err)

	return fake.NewFakeClient(testTokensSecret, testConfig)
}

func createDynaKube(activeGateEnabled bool) dynakube.DynaKube {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-namespace",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			Proxy: &value.Source{
				Value: "http://test-proxy:8080",
			},
			TelemetryIngest: &telemetryingest.Spec{},
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

	if activeGateEnabled {
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.RoutingCapability.DisplayName,
			},
		}
	}

	return dk
}

func reconcile(t *testing.T, ctx context.Context, clt client.WithWatch, dk dynakube.DynaKube) (dtEndpoint string, noProxy string) {
	er := endpoint.NewReconciler(clt, clt, &dk)
	err := er.Reconcile(ctx)
	require.NoError(t, err)

	var apiEndpointConfigMap corev1.ConfigMap
	err = clt.Get(ctx, types.NamespacedName{Name: consts.OtlpAPIEndpointConfigMapName, Namespace: dk.Namespace}, &apiEndpointConfigMap)
	require.NoError(t, err)
	var ok bool
	dtEndpoint, ok = apiEndpointConfigMap.Data["DT_ENDPOINT"]
	require.True(t, ok)

	sr := statefulset.NewReconciler(clt, clt, &dk)
	err = sr.Reconcile(ctx)
	require.NoError(t, err)

	var otelcSts appsv1.StatefulSet
	err = clt.Get(ctx, types.NamespacedName{Name: dk.OtelCollectorStatefulsetName(), Namespace: dk.Namespace}, &otelcSts)
	require.NoError(t, err)
	require.NotEmpty(t, otelcSts.Spec.Template.Spec.Containers)

	envs := otelcSts.Spec.Template.Spec.Containers[0].Env
	for _, env := range envs {
		if env.Name == "NO_PROXY" {
			noProxy = env.Value

			return dtEndpoint, noProxy
		}
	}

	require.FailNow(t, "failed to find env var NO_PROXY")

	return "", ""
}
