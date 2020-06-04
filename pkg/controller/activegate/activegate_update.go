package activegate

import (
	"context"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileActiveGate) findOutdatedPods(logger logr.Logger, instance *dynatracev1alpha1.ActiveGate) ([]v1.Pod, error) {
	secret, err := r.getTokenSecret(instance)
	if err != nil {
		logger.Error(err, "failed to retrieve token secret")
		return nil, err
	}

	dtClient, err := builder.BuildDynatraceClient(r.client, instance, secret)
	if err != nil {
		logger.Error(err, err.Error())
		return nil, err
	}

	pods, err := r.findPods(instance)
	if err != nil {
		logger.Error(err, "failed to list pods")
		return nil, err
	}

	var outdatedPods []v1.Pod

	for _, pod := range pods {
		networkZone := DEFAULT_NETWORK_ZONE
		if instance.Spec.NetworkZone != "" {
			networkZone = instance.Spec.NetworkZone
		}

		activegates, err := dtClient.QueryOutdatedActiveGates(dtclient.ActiveGateQuery{
			Hostname:       pod.Spec.Hostname,
			NetworkAddress: pod.Status.HostIP,
			NetworkZone:    networkZone,
		})
		if err != nil {
			logger.Error(err, "failed to query activegates")
			return nil, err
		}
		if len(activegates) > 0 {
			outdatedPods = append(outdatedPods, pod)
		}
	}

	return outdatedPods, nil
}

func (r *ReconcileActiveGate) findPods(instance *dynatracev1alpha1.ActiveGate) ([]v1.Pod, error) {
	podList := &v1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.Labels),
	}
	err := r.client.List(context.TODO(), podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

const (
	DEFAULT_NETWORK_ZONE = "default"
)
