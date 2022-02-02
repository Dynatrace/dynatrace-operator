package nodes

import (
	"context"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *NodesController) determineDynakubeForNode(nodeName string) (*dynatracev1beta1.DynaKube, error) {
	dkList, err := r.getOneAgentList()
	if err != nil {
		return nil, err
	}

	return r.filterOneAgentFromList(dkList, nodeName), nil
}

func (r *NodesController) getOneAgentList() (*dynatracev1beta1.DynaKubeList, error) {
	watchNamespace := os.Getenv("POD_NAMESPACE")

	var dkList dynatracev1beta1.DynaKubeList
	err := r.client.List(context.TODO(), &dkList, client.InNamespace(watchNamespace))
	if err != nil {
		return nil, err
	}

	return &dkList, nil
}

func (r *NodesController) filterOneAgentFromList(dkList *dynatracev1beta1.DynaKubeList,
	nodeName string) *dynatracev1beta1.DynaKube {

	for _, dynakube := range dkList.Items {
		items := dynakube.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dynakube
		}
	}

	return nil
}
