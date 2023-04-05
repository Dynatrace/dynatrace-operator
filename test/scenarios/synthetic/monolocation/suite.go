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

type stepContext struct {
	t                *testing.T
	config           *envconf.Config
	dynaKube         dynatracev1beta1.DynaKube
	pod              corev1.Pod
	containerLogSpec corev1.PodLogOptions
}

var config tenant.Secret

func newFeature(t *testing.T) features.Feature {
	config = tenant.GetSingleTenantSecret(t)

	builder := features.New("synthetic capability with single loc")
	builder.Setup(requireSyntheticLoc)
	builder.Setup(requireSyntheticBrowserMonitor)

	ctx := &stepContext{}
	builder.Setup(ctx.initStepContext)

	gateDynaKube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(config.ApiUrl).
		WithActiveGate().
		Build()
	assess.InstallDynatraceWithTeardown(builder, &config, gateDynaKube)
	builder.Assess(
		"observability activegate deployed",
		ctx.requireStepContext(gateDynaKube, consts.MultiActiveGateName))
	builder.Assess("observability activegate running", ctx.requireObservabilityFocusedActiveGate)

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
		ctx.requireStepContext(synDynaKube, capability.SyntheticName))
	builder.Assess("synthetic activegate running", ctx.requireSyntheticFocusedActiveGate)
	builder.Assess("vuc running", ctx.requireOperableVuc)
	builder.Assess("visit completed", ctx.requireSyntheticVisitCompleted)

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

func (c *stepContext) initStepContext(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	c.t = t
	c.config = cfg

	return ctx
}

func (c *stepContext) requireStepContext(dynaKube dynatracev1beta1.DynaKube, component string) features.Func {
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

		c.dynaKube = dynaKube
		c.pod = pods.Items[0]

		return ctx
	}
}

func (c *stepContext) requireObservabilityFocusedActiveGate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const activatedModulesLogMsg = `Active:([
[:blank:]]+(kubernetes_monitoring|odin_collector|metrics_ingest)){3}[
[:blank:]]+Lifecycle[[:blank:]]+listeners:`
	c.containerLogSpec.Container = consts.ActiveGateContainerName

	activatedModulesLogMsgRegexp := regexp.MustCompile(activatedModulesLogMsg)
	require.Regexp(
		t,
		activatedModulesLogMsgRegexp,
		c.requireContainerLogToMatch(ctx, activatedModulesLogMsgRegexp),
		"on-service status for observability ActiveGate found in log")

	return ctx
}

func (c *stepContext) requireContainerLogToMatch(
	ctx context.Context,
	regexp *regexp.Regexp,
) string {
	const (
		logReadSeriesDuration = 3 * time.Minute
		logReadPeriod         = 10 * time.Second
	)

	var log string
	matches := func() bool {
		log = c.requireContainerLog(ctx)
		return regexp.MatchString(log)
	}

	require.Eventually(
		c.t,
		matches,
		logReadSeriesDuration,
		logReadPeriod)

	c.t.Logf(
		"%s/%s log:\n%s",
		c.pod.Name,
		c.containerLogSpec.Container,
		log)
	return log
}

func (c *stepContext) requireContainerLog(ctx context.Context) string {
	client, err := kubernetes.NewForConfig(
		c.config.Client().Resources().GetConfig())
	require.NoError(c.t, err, "k8s client created")

	logStream, err := client.CoreV1().
		Pods(c.dynaKube.Namespace).
		GetLogs(
			c.pod.Name,
			&c.containerLogSpec).
		Stream(ctx)
	require.NoError(c.t, err, "log streamified")

	return logs.RequireContent(c.t, logStream)
}

func (c *stepContext) requireSyntheticFocusedActiveGate(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const boundLocationLogMsg = `Setting synthetic private location id to %d
Applying default config
Enabling selected features: %s`
	c.containerLogSpec.Container = consts.ActiveGateContainerName

	boundLocationLogMsgRegexp, err := regexp.Compile(
		fmt.Sprintf(
			boundLocationLogMsg,
			int64(dynakube.SyntheticLocationOrdinal(c.dynaKube)),
			capability.SyntheticActiveGateEnvCapabilities))
	require.NoError(t, err, "regexp compiled")
	require.Regexp(
		t,
		boundLocationLogMsgRegexp,
		c.requireContainerLogToMatch(ctx, boundLocationLogMsgRegexp),
		"on-service status for synthetic ActiveGate found in log")

	return ctx
}

func (c *stepContext) requireOperableVuc(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	const activeLogMsg = `VUC state changed to: Running(\\z|
)`
	c.containerLogSpec.Container = consts.SyntheticContainerName

	activeLogMsgRegexp := regexp.MustCompile(activeLogMsg)
	require.Regexp(
		t,
		activeLogMsg,
		c.requireContainerLogToMatch(ctx, activeLogMsgRegexp),
		"VUC running status found in log")

	return ctx
}

func (c *stepContext) requireSyntheticVisitCompleted(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
		log = c.requireVucBrowserLog(ctx)
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

func (c *stepContext) requireVucBrowserLog(ctx context.Context) string {
	const log = "/var/log/dynatrace/synthetic/vuc-browser.log"

	logReadResult, err := pod.Exec(
		ctx,
		c.config.Client().Resources(),
		c.pod,
		consts.SyntheticContainerName,
		shell.ReadFile(log)...)
	require.NoError(c.t, err, "VUC browser log read")

	return logReadResult.StdOut.String()
}
