//go:build e2e

package monolocation

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func newFeature(t *testing.T) features.Feature {
	tenantSecret := tenant.GetSingleTenantSecret(t)
	requireSyntheticLoc(t, tenantSecret)
	requireSyntheticBrowserMonitor(t, tenantSecret)

	builder := features.New("synthetic capability with single loc")

	agDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(tenantSecret.ApiUrl).
		WithActiveGate().
		Build()
	assess.InstallDynatraceWithTeardown(builder, &tenantSecret, agDynakube)
	builder.Assess("observability activegate deployed", activegate.WaitForStatefulSet(&agDynakube, consts.MultiActiveGateName))
	builder.Assess("observability activegate running", requireObservabilityFocusedActiveGate(agDynakube))

	synDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name("private-loc").
		WithSyntheticLocation(tenantSecret.SyntheticLocEntityId).
		ApiUrl(tenantSecret.ApiUrl).
		Build()
	assess.InstallDynakubeWithTeardown(builder, &tenantSecret, synDynakube)
	builder.Assess("synthetic loc deployed", activegate.WaitForStatefulSet(&synDynakube, capability.SyntheticName))
	builder.Assess("synthetic activegate running", requireSyntheticFocusedActiveGate(synDynakube))
	builder.Assess("vuc running", requireOperableVuc(synDynakube))
	builder.Assess("visit completed", requireSyntheticVisitCompleted(synDynakube, tenantSecret))

	return builder.Feature()
}

func requireSyntheticLoc(t *testing.T, secret tenant.Secret) {
	if secret.SyntheticLocEntityId == "" {
		t.Skip("suite skipped for the undefined synthetic location")
	}
}

func requireSyntheticBrowserMonitor(t *testing.T, secret tenant.Secret) {
	if secret.SyntheticBrowserMonitorEntityId == "" {
		t.Skip("suite skipped for the undefined synthetic browser monitor")
	}
}

func requireObservabilityFocusedActiveGate(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		const activatedModulesLogMsg = `Active:([
[:blank:]]+(kubernetes_monitoring|odin_collector|metrics_ingest)){3}[
[:blank:]]+Lifecycle[[:blank:]]+listeners:`
		activatedModulesLogMsgRegexp := regexp.MustCompile(activatedModulesLogMsg)
		requireContainerLogToMatch(ctx, t, cfg,
			activatedModulesLogMsgRegexp,
			testDynakube.Namespace,
			activegate.GetActiveGatePodName(&testDynakube, consts.MultiActiveGateName),
			consts.ActiveGateContainerName,
		)
		return ctx
	}
}

func requireSyntheticFocusedActiveGate(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		const boundLocationLogMsg = `Setting synthetic private location id to %d
Applying default config
Enabling selected features: %s`
		boundLocationLogMsgRegexp, err := regexp.Compile(
			fmt.Sprintf(
				boundLocationLogMsg,
				int64(dynakube.SyntheticLocationOrdinal(testDynakube)),
				capability.SyntheticActiveGateEnvCapabilities))
		require.NoError(t, err, "regexp compiled")
		requireContainerLogToMatch(ctx, t, cfg,
			boundLocationLogMsgRegexp,
			testDynakube.Namespace,
			activegate.GetActiveGatePodName(&testDynakube, capability.SyntheticName),
			consts.ActiveGateContainerName,
		)
		return ctx
	}
}

func requireOperableVuc(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		const activeLogMsg = `VUC state changed to: Running(\\z|
)`
		activeLogMsgRegexp := regexp.MustCompile(activeLogMsg)
		requireContainerLogToMatch(ctx, t, cfg,
			activeLogMsgRegexp,
			testDynakube.Namespace,
			activegate.GetActiveGatePodName(&testDynakube, capability.SyntheticName),
			consts.SyntheticContainerName,
		)
		return ctx
	}
}

func requireSyntheticVisitCompleted(testDynakube dynatracev1beta1.DynaKube, secret tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		const (
			logReadSeriesDuration = 7 * time.Minute
			logReadPeriod         = 20 * time.Second
		)

		regexp, err := regexp.Compile(
			fmt.Sprintf(
				"Visit \\[[[:digit:]]+/%s/[[:digit:]]+/[[:digit:]]+\\] completed with state TEST_COMPLETED",
				secret.SyntheticBrowserMonitorEntityId))
		require.NoError(t, err, "regexp compiled")

		var log string
		matches := func() bool {
			log = requireVucBrowserLog(ctx, t, cfg, testDynakube)
			return regexp.MatchString(log)
		}

		require.Eventually(
			t,
			matches,
			logReadSeriesDuration,
			logReadPeriod)
		return ctx
	}
}

func requireVucBrowserLog(ctx context.Context, t *testing.T, cfg *envconf.Config, testDynakube dynatracev1beta1.DynaKube) string {
	const log = "/var/log/dynatrace/synthetic/vuc-browser.log"

	agPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      activegate.GetActiveGatePodName(&testDynakube, capability.SyntheticName),
			Namespace: testDynakube.Namespace,
		},
	}

	logReadResult, err := pod.Exec(
		ctx,
		cfg.Client().Resources(),
		agPod,
		consts.SyntheticContainerName,
		shell.ReadFile(log)...)
	require.NoError(t, err, "VUC browser log read")

	return logReadResult.StdOut.String()
}

func requireContainerLogToMatch( //nolint:revive
	ctx context.Context,
	t *testing.T,
	cfg *envconf.Config,
	regexp *regexp.Regexp,
	namespace, pod, container string,
) {
	const (
		logReadSeriesDuration = 3 * time.Minute
		logReadPeriod         = 10 * time.Second
	)

	var log string
	matches := func() bool {
		log = logs.ReadLog(ctx, t, cfg, namespace, pod, container)
		return regexp.MatchString(log)
	}

	require.Eventually(
		t,
		matches,
		logReadSeriesDuration,
		logReadPeriod)
}
