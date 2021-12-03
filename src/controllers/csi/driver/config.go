package csidriver

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	log               = logger.NewDTLogger().WithName("csi-driver")
	memoryUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "memory_usage",
		Help:      "Memory usage of the csi driver in bytes",
	})
	agentsVersionsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "agent_versions",
		Help:      "Number of an agent version currently mounted by the CSI driver",
	}, []string{"version"})
	memoryMetricTick = 5000 * time.Millisecond
)

func init() {
	metrics.Registry.MustRegister(memoryUsageMetric)
	metrics.Registry.MustRegister(agentsVersionsMetric)
}
