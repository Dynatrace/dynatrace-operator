package appvolumes

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	log                  = logger.NewDTLogger().WithName("csi-driver.app")
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
