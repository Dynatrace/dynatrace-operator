package common

import "github.com/Dynatrace/dynatrace-operator/src/webhook"

const (
	DynaMetrics = "dynametrics"

	KubjectNamePrefix = "dynatrace-metrics"

	HttpsServicePortName = "https"
	HttpsServicePort     = 6443
	HttpServicePortName  = "http"
	HttpContainerPort    = 8080

	DynaMetricClusterRoleName = "dynatrace-metric-server"

	ApiServiceGroup        = "external.metrics.k8s.io"
	ApiServiceVersion      = "v1beta1"
	ApiServiceVersionGroup = ApiServiceVersion + "." + ApiServiceGroup

	ControlledByDynaKubeAnnotation = webhook.InjectionInstanceLabel
)
