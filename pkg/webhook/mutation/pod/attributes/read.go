package attributes

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (attrs *Pod) readMetadataAnnotations(request mutator.BaseRequest) {
	attrs.applyEnrichmentRules(request.Namespace, request.DynaKube)
	attrs.readNamespaceAnnotationAttributes(request.Namespace)
	attrs.readPodAnnotationAttributes(*request.Pod)
}

// collect attributes from pod and namespace "metadata.dynatrace.com/" annotations
func (attrs *Pod) readNamespaceAnnotationAttributes(namespace corev1.Namespace) {
	for key, value := range namespace.Annotations {
		if after, ok := strings.CutPrefix(key, metadataenrichment.Prefix); ok {
			attrs.namespaceAnnotations[after] = value
		}
	}
}

// collect attributes from pod and namespace "metadata.dynatrace.com/" annotations
func (attrs *Pod) readPodAnnotationAttributes(pod corev1.Pod) {
	// pod annotations take precedence over namespace annotations
	for key, value := range pod.Annotations {
		if after, ok := strings.CutPrefix(key, metadataenrichment.Prefix); ok {
			attrs.podAnnotations[after] = value
		}
	}
}

func (attrs *Pod) applyEnrichmentRules(namespace corev1.Namespace, dk dynakube.DynaKube) {
	for _, rule := range dk.Status.MetadataEnrichment.Rules {
		var (
			valueFromNamespace string
			exists             bool
		)

		switch rule.Type {
		case metadataenrichment.LabelRule, metadataenrichment.K8sLabelRule:
			valueFromNamespace, exists = namespace.Labels[rule.Source]
		case metadataenrichment.AnnotationRule, metadataenrichment.K8sAnnotationRule:
			valueFromNamespace, exists = namespace.Annotations[rule.Source]
		case metadataenrichment.CustomRule:
			valueFromNamespace = rule.Source
			exists = true
		}

		if exists {
			if len(rule.Target) > 0 {
				attrs.rulesPropagate[rule.Target] = valueFromNamespace
			} else {
				keyPart := rule.Source
				if rule.Type == metadataenrichment.CustomRule {
					keyPart = resourceattributes.SanitizeKey(rule.Source)
				}

				attrs.rules[metadataenrichment.GetEmptyTargetEnrichmentKey(string(rule.Type), keyPart)] = valueFromNamespace
			}
		}
	}
}

func (attrs *Pod) readWorkloadInfoAttributes(ctx context.Context, request mutator.BaseRequest, client client.Client) error {
	workloadInfo, err := workload.FindRootOwnerOfPod(ctx, client, request)
	if err != nil {
		return errors.WithStack(err)
	}

	attrs.workloadInfo[K8sWorkloadKindAttr] = workloadInfo.Kind
	attrs.workloadInfo[K8sWorkloadNameAttr] = workloadInfo.Name

	return nil
}

func (attrs *Pod) readPodAttributes(request mutator.BaseRequest) {
	attrs.podEnvVars = append(attrs.podEnvVars,
		corev1.EnvVar{Name: K8sPodNameEnv, ValueFrom: k8senv.NewSourceForField("metadata.name")},
		corev1.EnvVar{Name: K8sPodUIDEnv, ValueFrom: k8senv.NewSourceForField("metadata.uid")},
		corev1.EnvVar{Name: K8sNodeNameEnv, ValueFrom: k8senv.NewSourceForField("spec.nodeName")},
	)

	attrs.podInfo[K8sPodNameAttr] = k8senv.NewRef(K8sPodNameEnv)
	attrs.podInfo[K8sPodUIDAttr] = k8senv.NewRef(K8sPodUIDEnv)
	attrs.podInfo[K8sNodeNameAttr] = k8senv.NewRef(K8sNodeNameEnv)
	attrs.podInfo[K8sNamespaceNameAttr] = request.Pod.Namespace

	attrs.clusterInfo[K8sClusterUIDAttr] = request.DynaKube.Status.KubeSystemUUID
	attrs.clusterInfo[K8sClusterNameAttr] = request.DynaKube.Status.KubernetesClusterName
	attrs.clusterInfo[K8sDTClusterEntityAttr] = request.DynaKube.Status.KubernetesClusterMEID
}
