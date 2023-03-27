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
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type stepParams struct {
	context context.Context
	t       *testing.T
	config  *envconf.Config
}

type stepContext struct {
	dynaKube         dynatracev1beta1.DynaKube
	pod              corev1.Pod
	containerLogSpec corev1.PodLogOptions
}

type contextValueKey int

var (
	config              tenant.Secret
	universalStepParams stepParams

	stepContextKey = contextValueKey(1)
)

func (params stepParams) stepContext() *stepContext {
	return universalStepParams.context.
		Value(stepContextKey).(*stepContext)
}

func newFeature(t *testing.T) features.Feature {
	config = tenant.GetSingleTenantSecret(t)

	builder := features.New("synthetic capability with single loc")
	builder.Setup(requireSyntheticLoc)
	builder.Setup(requireSyntheticBrowserMonitor)
	builder.Setup(copyStepParams)

	gateDynaKube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(config.ApiUrl).
		WithActiveGate().
		Build()
	assess.InstallDynatraceWithTeardown(builder, &config, gateDynaKube)
	builder.Assess(
		"observability activegate deployed",
		requireStepContext(gateDynaKube, consts.MultiActiveGateName))
	builder.Assess("observability activegate running", requireObservabilityFocusedActiveGate)

	synDynaKube := dynakube.NewBuilder().
		Name("private-loc").
		Namespace(gateDynaKube.Namespace).
		WithSyntheticLocation(config.SyntheticLocEntityId).
		ApiUrl(config.ApiUrl).
		Tokens(gateDynaKube.Name).
		Build()
	assess.InstallDynakubeWithTeardown(builder, nil, synDynaKube)
	builder.Assess(
		"synthetic loc deployed",
		requireStepContext(synDynaKube, capability.SyntheticName))
	builder.Assess("synthetic activegate running", requireSyntheticFocusedActiveGate)
	builder.Assess("vuc running", requireOperableVuc)
	builder.Assess("visit completed", requireSyntheticVisitCompleted)

	return builder.Feature()
}

func requireSyntheticLoc(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	if config.SyntheticLocEntityId == "" {
		t.Skip("suite skipped for the undefined synthetic location")
	}
	return ctx
}

func requireSyntheticBrowserMonitor(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	if config.SyntheticBrowserMonitorEntityId == "" {
		t.Skip("suite skipped for the undefined synthetic browser monitor")
	}
	return ctx
}

func copyStepParams(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	universalStepParams = stepParams{
		context: context.WithValue(ctx, stepContextKey, &stepContext{}),
		t:       t,
		config:  cfg,
	}

	return ctx
}

func requireStepContext(dynaKube dynatracev1beta1.DynaKube, component string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		pods := pod.GetPodsForOwner(
			ctx,
			t,
			cfg.Client().Resources(),
			dynaKube.Name+"-"+component,
			dynaKube.Namespace)
		require.Equalf(
			t,
			len(pods.Items),
			1,
			"unique %s pod deployed",
			component)

		universalStepParams.stepContext().dynaKube = dynaKube
		universalStepParams.stepContext().pod = pods.Items[0]

		return ctx
	}
}

func requireObservabilityFocusedActiveGate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const activatedModulesLogMsg = `Active:([
[:blank:]]+(kubernetes_monitoring|odin_collector|metrics_ingest)){3}[
[:blank:]]+Lifecycle[[:blank:]]+listeners:`
	universalStepParams.stepContext().containerLogSpec.Container = consts.ActiveGateContainerName

	activatedModulesLogMsgRegexp := regexp.MustCompile(activatedModulesLogMsg)
	require.Regexp(
		t,
		activatedModulesLogMsgRegexp,
		requireContainerLogToMatch(activatedModulesLogMsgRegexp),
		"on-service status for observability ActiveGate found in log")

	return ctx
}

func requireContainerLogToMatch(regexp *regexp.Regexp) string {
	const (
		logReadSeriesDuration = 3 * time.Minute
		logReadPeriod         = 10 * time.Second
	)

	var log string
	matches := func() bool {
		log = requireContainerLog()
		return regexp.MatchString(log)
	}

	require.Eventually(
		universalStepParams.t,
		matches,
		logReadSeriesDuration,
		logReadPeriod)

	universalStepParams.t.Logf(
		"%s/%s log:\n%s",
		universalStepParams.stepContext().pod.Name,
		universalStepParams.stepContext().containerLogSpec.Container,
		log)
	return log
}

func requireContainerLog() string {
	client, err := kubernetes.NewForConfig(
		universalStepParams.config.Client().Resources().GetConfig())
	require.NoError(universalStepParams.t, err, "k8s client created")

	logStream, err := client.CoreV1().
		Pods(universalStepParams.stepContext().dynaKube.Namespace).
		GetLogs(
			universalStepParams.stepContext().pod.Name,
			&universalStepParams.stepContext().containerLogSpec).
		Stream(universalStepParams.context)
	require.NoError(universalStepParams.t, err, "log streamified")

	return logs.RequireContent(universalStepParams.t, logStream)
}

func requireSyntheticFocusedActiveGate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const boundLocationLogMsg = `Setting synthetic private location id to %d
Applying default config
Enabling selected features: %s`
	universalStepParams.stepContext().containerLogSpec.Container = consts.ActiveGateContainerName

	boundLocationLogMsgRegexp, err := regexp.Compile(
		fmt.Sprintf(
			boundLocationLogMsg,
			int64(dynakube.SyntheticLocationOrdinal(universalStepParams.stepContext().dynaKube)),
			capability.SyntheticActiveGateEnvCapabilities))
	require.NoError(t, err, "regexp compiled")
	require.Regexp(
		t,
		boundLocationLogMsgRegexp,
		requireContainerLogToMatch(boundLocationLogMsgRegexp),
		"on-service status for synthetic ActiveGate found in log")

	return ctx
}

func requireOperableVuc(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const activeLogMsg = `VUC state changed to: Running(\\z|
)`
	universalStepParams.stepContext().containerLogSpec.Container = consts.SyntheticContainerName

	activeLogMsgRegexp := regexp.MustCompile(activeLogMsg)
	require.Regexp(
		t,
		activeLogMsg,
		requireContainerLogToMatch(activeLogMsgRegexp),
		"VUC running status found in log")

	return ctx
}

func requireSyntheticVisitCompleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const (
		logReadSeriesDuration = 7 * time.Minute
		logReadPeriod         = 20 * time.Second
	)

	regexp, err := regexp.Compile(
		fmt.Sprintf(
			"Visit \\[[[:digit:]]+/%s/[[:digit:]]+/[[:digit:]]+\\] completed with state TEST_COMPLETED",
			config.SyntheticBrowserMonitorEntityId))
	require.NoError(t, err, "regexp compiled")

	var log string
	matches := func() bool {
		log = requireVucBrowserLog()
		return regexp.MatchString(log)
	}

	require.Eventually(
		t,
		matches,
		logReadSeriesDuration,
		logReadPeriod)

	t.Logf("vuc-browser.log:\n%s", log)
	require.Regexp(t, regexp, log, "visit completed")

	return ctx
}

func requireVucBrowserLog() string {
	const log = "/var/log/dynatrace/synthetic/vuc-browser.log"

	logReadResult, err := pod.Exec(
		universalStepParams.context,
		universalStepParams.config.Client().Resources(),
		universalStepParams.stepContext().pod,
		consts.SyntheticContainerName,
		shell.ReadFile(log)...)
	require.NoError(universalStepParams.t, err, "VUC browser log read")

	return logReadResult.StdOut.String()
}
