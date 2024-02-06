package nodes

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (controller *Controller) determineDynakubeForNode(nodeName string) (*dynatracev1beta1.DynaKube, error) {
	dks, err := controller.getDynakubeList()
	if err != nil {
		return nil, err
	}

	return controller.filterDynakubeFromList(dks, nodeName), nil
}

func (controller *Controller) getDynakubeList() (*dynatracev1beta1.DynaKubeList, error) {
	var dynakubeList dynatracev1beta1.DynaKubeList

	err := controller.apiReader.List(context.TODO(), &dynakubeList, client.InNamespace(controller.podNamespace))
	if err != nil {
		return nil, err
	}

	return &dynakubeList, nil
}

func (controller *Controller) filterDynakubeFromList(dkList *dynatracev1beta1.DynaKubeList,
	nodeName string) *dynatracev1beta1.DynaKube {
	for _, dynakube := range dkList.Items {
		items := dynakube.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dynakube
		}
	}

	return nil
}
