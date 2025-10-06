package exporter

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	corev1 "k8s.io/api/core/v1"
)

// injector defines the interface for injecting signal specific OTLP env vars.
type injector interface {
	Inject(c *corev1.Container, apiURL string, override bool) bool
}

// traceInjector handles traces signal env var injection.
type traceInjector struct {
	cfg *otlpexporterconfiguration.OTLPExporterConfiguration
}

// isEnabled returns true if traces should be injected according to the configuration.
func (ti *traceInjector) isEnabled() bool {
	if ti == nil || ti.cfg == nil {
		return false
	}
	return ti.cfg.IsTracesEnabled()
}

// Inject performs the actual injection of trace env vars.
func (ti *traceInjector) Inject(c *corev1.Container, apiURL string, override bool) bool {
	if !ti.isEnabled() {
		return false
	}
	addEnvVarLiteralValue(c, OTLPTraceEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "traces"))
	addEnvVarLiteralValue(c, OTLPTraceProtocolEnv, "http/protobuf")

	return true
}

// metricsInjector handles metrics signal env var injection.
type metricsInjector struct {
	cfg *otlpexporterconfiguration.OTLPExporterConfiguration
}

func (mi *metricsInjector) isEnabled() bool {
	if mi == nil || mi.cfg == nil {
		return false
	}
	return mi.cfg.IsMetricsEnabled()
}

func (mi *metricsInjector) Inject(c *corev1.Container, apiURL string, override bool) bool {
	if !mi.isEnabled() {
		return false
	}

	addEnvVarLiteralValue(c, OTLPMetricsEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "metrics"))
	addEnvVarLiteralValue(c, OTLPMetricsProtocolEnv, "http/protobuf")

	return true
}

// logsInjector handles logs signal env var injection.
type logsInjector struct {
	cfg *otlpexporterconfiguration.OTLPExporterConfiguration
}

func (li *logsInjector) isEnabled() bool {
	if li == nil || li.cfg == nil {
		return false
	}
	return li.cfg.IsLogsEnabled()
}

func (li *logsInjector) Inject(c *corev1.Container, apiURL string, override bool) bool {
	if !li.isEnabled() {
		return false
	}

	addEnvVarLiteralValue(c, OTLPLogsEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "logs"))
	addEnvVarLiteralValue(c, OTLPLogsProtocolEnv, "http/protobuf")

	return true
}

func addEnvVarLiteralValue(c *corev1.Container, name string, value string) {
	c.Env = env.AddOrUpdate(c.Env, corev1.EnvVar{Name: name, Value: value})
}
