//go:build e2e

package logmonitoring

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	lmdaemonset "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/logmonsettings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("logmonitoring-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)
	secretConfig.APITokenNoSettings = "" // Always use more privileged token

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithLogMonitoring(),
		componentDynakube.WithLogMonitoringImageRefSpec(consts.LogMonitoringImageRepo, consts.LogMonitoringImageTag),
		componentDynakube.WithActiveGate(),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
	}

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		options = append(options, componentDynakube.WithAnnotations(map[string]string{
			exp.OAPrivilegedKey: "true",
		}))
	}

	testDynakube := *componentDynakube.New(options...)

	agCrt, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificate))
	require.NoError(t, err)

	agP12, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	require.NoError(t, err)

	agSecret := secret.New(consts.AgSecretName, testDynakube.Namespace,
		map[string][]byte{
			dynakube.TLSCertKey:                    agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", secret.Create(agSecret))

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", checkActiveGateContainer(&testDynakube))

	builder.Assess("log agent started", daemonset.WaitForDaemonset(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("log monitoring conditions", checkConditions(testDynakube.Name, testDynakube.Namespace, true))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	builder.WithTeardown("deleted ag secret", secret.Delete(agSecret))

	return builder.Feature()
}

func WithOptionalScopes(t *testing.T) features.Feature {
	builder := features.New("logmonitoring-with-optional-scopes")

	secretConfig := tenant.GetSingleTenantSecret(t)
	if secretConfig.APITokenNoSettings == "" {
		t.Skip("skipping test. no token with missing settings scopes provided")
	}

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithLogMonitoring(),
		componentDynakube.WithLogMonitoringImageRefSpec(consts.LogMonitoringImageRepo, consts.LogMonitoringImageTag),
		componentDynakube.WithActiveGate(),
	}

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		options = append(options, componentDynakube.WithAnnotations(map[string]string{
			exp.OAPrivilegedKey: "true",
		}))
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.InstallWithoutSettingsScopes(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", checkActiveGateContainer(&testDynakube))

	builder.Assess("log agent started", daemonset.WaitForDaemonset(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("log monitoring conditions with disabled scopes", checkConditions(testDynakube.Name, testDynakube.Namespace, false))

	builder.Assess("update token secret", tenant.CreateTenantSecret(secretConfig.APIToken, secretConfig.DataIngestToken, testDynakube.Name, testDynakube.Namespace))

	builder.Assess("trigger reconcile", triggerDaemonSetReconcile(testDynakube))

	builder.Assess("log agent restarted", daemonset.WaitForDaemonset(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("log monitoring conditions with enabled scopes", checkConditions(testDynakube.Name, testDynakube.Namespace, true))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

func checkActiveGateContainer(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.WithNamespace(dk.Namespace).Get(ctx, activegate.GetActiveGatePodName(dk, "activegate"), dk.Namespace, &activeGatePod))

		require.NotNil(t, activeGatePod.Spec)
		require.NotEmpty(t, activeGatePod.Spec.Containers)

		return ctx
	}
}

func checkConditions(name string, namespace string, scopesEnabled bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		dk := &dynakube.DynaKube{}
		err := envConfig.Client().Resources().Get(ctx, name, namespace, dk)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), configsecret.LmcConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		condition = meta.FindStatusCondition(*dk.Conditions(), lmdaemonset.ConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.DaemonSetSetCreatedReason, condition.Reason)

		if scopesEnabled {
			assert.True(t, meta.IsStatusConditionTrue(dk.Status.Conditions, logmonsettings.ConditionType))
		} else {
			assert.True(t, meta.IsStatusConditionFalse(dk.Status.Conditions, logmonsettings.ConditionType))
		}

		for _, conditionType := range dtclient.OptionalScopes {
			hasScope := conditions.IsOptionalScopeAvailable(dk, conditionType)
			assert.Equalf(t, scopesEnabled, hasScope, "expected %s condition to be %t", conditionType, scopesEnabled)
		}

		return ctx
	}
}

func triggerDaemonSetReconcile(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		logMonitoring := daemonset.NewQuery(ctx, resources, client.ObjectKey{Name: dk.LogMonitoring().GetDaemonSetName(), Namespace: dk.Namespace})

		logMonDaemonSet, err := logMonitoring.Get()
		require.NoError(t, err)
		prevGeneration := logMonDaemonSet.Generation

		require.NoError(t, resources.Get(ctx, dk.Name, dk.Namespace, &dk))
		// Force reconciliation by simulating the passage of time
		dk.Status.DynatraceAPI.LastTokenScopeRequest.Time = dk.Status.DynatraceAPI.LastTokenScopeRequest.Add(-2 * dk.APIRequestThreshold())
		expireLastTransitionTime(&dk, "MonitoredEntity")
		expireLastTransitionTime(&dk, logmonsettings.ConditionType)
		require.NoError(t, resources.UpdateStatus(ctx, &dk))

		// Verify that the operator picked up the update
		err = wait.For(func(ctx context.Context) (bool, error) {
			logMonDaemonSet, err := logMonitoring.Get()

			return logMonDaemonSet.Status.ObservedGeneration != prevGeneration, err
		}, wait.WithTimeout(1*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}

func expireLastTransitionTime(dk *dynakube.DynaKube, conditionType string) {
	cond := meta.FindStatusCondition(dk.Status.Conditions, conditionType)
	if cond == nil {
		return
	}
	cond.LastTransitionTime.Time = cond.LastTransitionTime.Add(-2 * dk.APIRequestThreshold())
	meta.SetStatusCondition(&dk.Status.Conditions, *cond)
}
