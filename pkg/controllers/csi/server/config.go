package csiserver

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	log               = logd.Get().WithName("csi-driver")
	memoryUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "memory_usage",
		Help:      "Memory usage of the csi driver in bytes",
	})
	memoryMetricTick = 5000 * time.Millisecond
)

func init() {
	metrics.Registry.MustRegister(memoryUsageMetric)
}
