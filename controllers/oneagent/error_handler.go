package oneagent

import (
	"errors"
	"net/http"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func handlePodListError(logger logr.Logger, err error, listOps []client.ListOption) {
	logger.Error(err, "failed to list pods", "listops", listOps)
}

func handleAgentVersionForIPError(err error, instance *dynatracev1alpha1.DynaKube, pod corev1.Pod, instanceStatus *dynatracev1alpha1.OneAgentInstance) error {
	if err != nil {
		var serr dtclient.ServerError
		if ok := errors.As(err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			return err
		}
		// use last know version if available
		if i, ok := instance.Status.OneAgent.Instances[pod.Spec.NodeName]; ok && instanceStatus != nil {
			instanceStatus.Version = i.Version
		}
	}
	return nil
}
