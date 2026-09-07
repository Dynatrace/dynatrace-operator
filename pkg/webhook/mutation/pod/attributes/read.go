// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package attributes

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (attrs *Pod) readMetadataAnnotations(request mutator.BaseRequest, workloadInfo *workload.Info) {
	attrs.applyEnrichmentRules(request.DynaKube.Status.MetadataEnrichment.Rules, &request.Namespace, workloadInfo, request.Pod)
	attrs.readNamespaceAnnotationAttributes(&request.Namespace)
	attrs.readWorkloadAnnotationAttributes(workloadInfo)
	attrs.readPodAnnotationAttributes(request.Pod)
}

func (attrs *Pod) readNamespaceAnnotationAttributes(namespace *corev1.Namespace) {
	readPrefixedAnnotations(attrs.namespaceAnnotations, namespace.Annotations)
}

func (attrs *Pod) readWorkloadAnnotationAttributes(workloadInfo *workload.Info) {
	readPrefixedAnnotations(attrs.workloadAnnotations, workloadInfo.Annotations)
}

func (attrs *Pod) readPodAnnotationAttributes(pod *corev1.Pod) {
	readPrefixedAnnotations(attrs.podAnnotations, pod.Annotations)
}

// collect attributes from namespace, workload and pod "metadata.dynatrace.com/" annotations
func readPrefixedAnnotations(attributes, annotations map[string]string) {
	for key, value := range annotations {
		if after, ok := strings.CutPrefix(key, metadataenrichment.Prefix); ok {
			attributes[after] = value
		}
	}
}

func (attrs *Pod) applyEnrichmentRules(rules []metadataenrichment.Rule, namespace *corev1.Namespace, workloadInfo *workload.Info, pod *corev1.Pod) {
	for _, rule := range rules {
		var (
			value  string
			exists bool
		)

		switch rule.Type {
		case metadataenrichment.LabelRule, metadataenrichment.K8sNamespaceLabelRule:
			value, exists = namespace.Labels[rule.Source]
		case metadataenrichment.AnnotationRule, metadataenrichment.K8sNamespaceAnnotationRule:
			value, exists = namespace.Annotations[rule.Source]
		case metadataenrichment.K8sWorkloadLabelRule:
			value, exists = workloadInfo.Labels[rule.Source]
		case metadataenrichment.K8sWorkloadAnnotationRule:
			value, exists = workloadInfo.Annotations[rule.Source]
		case metadataenrichment.K8sPodLabelRule:
			value, exists = pod.Labels[rule.Source]
		case metadataenrichment.K8sPodAnnotationRule:
			value, exists = pod.Annotations[rule.Source]
		case metadataenrichment.CustomRule:
			value = rule.Source
			exists = true
		}

		if exists {
			if rule.Target == "" {
				// The last rule without a target to resolve to a value wins. This inconsistency is intentional to not introduce a behavior change for users upgrading from <1.10.
				// Target can only be empty for legacy schema rules.
				attrs.rules[metadataenrichment.GetEmptyTargetEnrichmentKey(string(rule.Type), rule.Source)] = value
			} else if _, ok := attrs.rules[rule.Target]; !ok {
				// The first rule to resolve to a value wins, regardless of type.
				// This only is valid if the order of the rules remains unchanged from the API response.
				attrs.rules[rule.Target] = value
			}
		}
	}
}

func (attrs *Pod) readWorkloadInfoAttributes(ctx context.Context, request mutator.BaseRequest, client client.Client) (*workload.Info, error) {
	workloadInfo, err := workload.FindRootOwnerOfPod(ctx, client, request)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	attrs.workloadInfo[K8sWorkloadKindAttr] = workloadInfo.Kind
	attrs.workloadInfo[K8sWorkloadNameAttr] = workloadInfo.Name

	return workloadInfo, nil
}

func (attrs *Pod) readPodAttributes(request mutator.BaseRequest) {
	attrs.podEnvVars = append(
		attrs.podEnvVars,
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
