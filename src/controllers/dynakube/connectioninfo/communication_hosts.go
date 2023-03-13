package connectioninfo

import (
	"context"
	"encoding/json"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCommunicationHosts(ctx context.Context, client client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) ([]dtclient.CommunicationHost, error) {
	connectionInfoQuery := kubeobjects.NewConfigMapQuery(ctx, client, apiReader, log)
	oneAgentConnectionInfo, err := connectionInfoQuery.Get(types.NamespacedName{Name: dynakube.OneAgentConnectionInfoConfigMapName(), Namespace: dynakube.Namespace})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query configmap")
	}
	communicationHostsString, err := kubeobjects.ExtractField(&oneAgentConnectionInfo, CommunicationHosts)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to extract %s field of %s configmap", CommunicationHosts, dynakube.OneAgentConnectionInfoConfigMapName())
	}
	var communicationHosts []dtclient.CommunicationHost
	if err := json.Unmarshal([]byte(communicationHostsString), &communicationHosts); err != nil {
		return nil, errors.WithMessagef(err, "failed to decode %s field of %s configmap", CommunicationHosts, dynakube.OneAgentConnectionInfoConfigMapName())
	}
	return communicationHosts, nil
}
