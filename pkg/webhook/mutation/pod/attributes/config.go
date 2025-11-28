package attributes

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

const (
	K8sNodeNameEnv = "K8S_NODE_NAME"
	K8sPodNameEnv  = "K8S_PODNAME"
	K8sPodUIDEnv   = "K8S_PODUID"

	// AnnotationWorkloadKind is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadKind = "metadata.dynatrace.com/k8s.workload.kind"
	// AnnotationWorkloadName is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadName = "metadata.dynatrace.com/k8s.workload.name"
)

var (
	log = logd.Get().WithName("metadata-attributes")
)
