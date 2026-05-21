package attributes

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	K8sContainerNameAttr = "k8s.container.name"
	K8sNodeNameEnv       = "K8S_NODE_NAME"
	K8sPodNameEnv        = "K8S_PODNAME"
	K8sPodUIDEnv         = "K8S_PODUID"

	K8sPodNameAttr       = "k8s.pod.name"
	K8sPodUIDAttr        = "k8s.pod.uid"
	K8sNodeNameAttr      = "k8s.node.name"
	K8sNamespaceNameAttr = "k8s.namespace.name"

	K8sClusterUIDAttr      = "k8s.cluster.uid"
	K8sClusterNameAttr     = "k8s.cluster.name"
	K8sDTClusterEntityAttr = "dt.entity.kubernetes_cluster"

	K8sWorkloadKindAttr = "k8s.workload.kind"
	K8sWorkloadNameAttr = "k8s.workload.name"
)

// sanitizeKeys returns a new map with all keys passed through resourceattributes.SanitizeKey.
// Entries whose key is empty after sanitization are dropped and logged.
func sanitizeKeys(ctx context.Context, attrs map[string]string) map[string]string {
	log := logd.FromContext(ctx)
	sanitized := make(map[string]string, len(attrs))

	for key, value := range attrs {
		sanitizedKey := resourceattributes.SanitizeKey(key)
		if sanitizedKey == "" {
			log.Info("dropping dynakube attribute: key is empty after sanitization", "originalKey", key)

			continue
		}

		sanitized[sanitizedKey] = value
	}

	return sanitized
}
