package appvolumes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	log                  = logger.Get().WithName("csi-appvolume")
	agentsVersionsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "agent_versions",
		Help:      "Number of an agent version currently mounted by the CSI driver",
	}, []string{"version"})
)

const Mode = "app"

func init() {
	metrics.Registry.MustRegister(agentsVersionsMetric)
}
