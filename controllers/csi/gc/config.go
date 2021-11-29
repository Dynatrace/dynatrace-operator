package csigc

import (
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	log = logger.NewDTLogger().WithName("csi-gc")

	reclaimedMemoryMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "gc_reclaimed",
		Help:      "Amount of memory reclaimed by the GC",
	})

	foldersRemovedMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "gc_folder_rmv",
		Help:      "Number of folders deleted by the GC",
	})

	gcRunsMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "gc_runs",
		Help:      "Number of GC runs",
	})
)

func init() {
	metrics.Registry.MustRegister(reclaimedMemoryMetric)
	metrics.Registry.MustRegister(foldersRemovedMetric)
	metrics.Registry.MustRegister(gcRunsMetric)
}
