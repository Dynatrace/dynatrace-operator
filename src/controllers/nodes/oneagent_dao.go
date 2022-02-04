package nodes

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (controller *NodesController) determineDynakubeForNode(nodeName string) (*dynatracev1beta1.DynaKube, error) {
	dks, err := controller.getDynakubeList()
	if err != nil {
		return nil, err
	}
	return controller.filterDynakubeFromList(dks, nodeName), nil
}

func (controller *NodesController) getDynakubeList() (*dynatracev1beta1.DynaKubeList, error) {
	var dynakubeList dynatracev1beta1.DynaKubeList
	err := controller.client.List(context.TODO(), &dynakubeList, client.InNamespace(controller.podNamespace))
	if err != nil {
		return nil, err
	}
	return &dynakubeList, nil
}

func (controller *NodesController) filterDynakubeFromList(dkList *dynatracev1beta1.DynaKubeList,
	nodeName string) *dynatracev1beta1.DynaKube {

	for _, dynakube := range dkList.Items {
		items := dynakube.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dynakube
		}
	}
	return nil
}
