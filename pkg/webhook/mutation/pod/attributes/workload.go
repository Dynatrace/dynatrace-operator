package attributes

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (attrs *PodAttributes) readWorkloadInfoAttributes(ctx context.Context, request dtwebhook.BaseRequest, client client.Client) error {
	workloadInfo, err := workload.FindRootOwnerOfPod(ctx, client, request)
	if err != nil {
		return errors.WithStack(err)
	}

	attrs.workloadInfo[K8sWorkloadKindAttr] = workloadInfo.Kind
	attrs.workloadInfo[K8sWorkloadNameAttr] = workloadInfo.Name

	return nil
}
