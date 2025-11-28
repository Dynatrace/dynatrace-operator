package attributes

import (
	"context"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetWorkloadInfoAttributes(attrs podattr.Attributes, ctx context.Context, request *mutator.BaseRequest, clt client.Client) (podattr.Attributes, error) {
	workloadInfo, err := workload.FindRootOwnerOfPod(ctx, clt, *request, log)
	if err != nil {
		return attrs, errors.WithStack(err)
	}

	attrs.WorkloadInfo = podattr.WorkloadInfo{
		WorkloadKind: workloadInfo.Kind,
		WorkloadName: workloadInfo.Name,
	}

	attrs = setDeprecatedWorkloadAttributes(attrs)

	setWorkloadAnnotations(request.Pod, workloadInfo)

	return attrs, nil
}

func setWorkloadAnnotations(pod *corev1.Pod, workload *workload.Info) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationWorkloadKind] = workload.Kind
	pod.Annotations[AnnotationWorkloadName] = workload.Name
}
