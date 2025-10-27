package statefulset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testTenantUUID     = "abc12345"
	testKubeSystemUUID = "12345"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	ctx := context.Background()

	dk := getTestDynakube()
	dk.Status = dynakube.DynaKubeStatus{
		ActiveGate: activegate.Status{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testTenantUUID,
			},
			VersionStatus: status.VersionStatus{
				ImageID: "thisismytenant.com/linux/activegate@sha256:312a5fafebb134371dc05e3e0ad00641bd44fde2a31b70dca5edbc708f2e76cb",
			},
		},
		KubeSystemUUID: testKubeSystemUUID,
	}
	dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, &dk)
	integrationtests.CreateKubernetesObject(t, ctx, clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName + activegate.AuthTokenSecretSuffix,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
	})

	mcap := capability.NewMultiCapability(&dk)
	reconciler := NewReconciler(clt, clt, &dk, mcap)

	err := reconciler.Reconcile(ctx)
	require.NoError(t, err)

	dk.Spec.ActiveGate.UseEphemeralVolume = true
	err = reconciler.Reconcile(ctx)
	require.NoError(t, err)
}
