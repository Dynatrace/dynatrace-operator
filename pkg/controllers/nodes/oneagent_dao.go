package nodes

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (controller *Controller) determineDynakubeForNode(ctx context.Context, nodeName string) (*dynakube.DynaKube, error) {
	dks, err := controller.getDynakubeList()
	if err != nil {
		return nil, err
	}

	return controller.filterDynakubeFromList(dks, nodeName), nil
}

func (controller *Controller) getDynakubeList(ctx context.Context) (*dynakube.DynaKubeList, error) {
	var dynakubeList dynakube.DynaKubeList

	err := controller.apiReader.List(ctx, &dynakubeList, client.InNamespace(controller.podNamespace))
	if err != nil {
		return nil, err
	}

	return &dynakubeList, nil
}

func (controller *Controller) filterDynakubeFromList(dkList *dynakube.DynaKubeList,
	nodeName string) *dynakube.DynaKube {
	for _, dk := range dkList.Items {
		items := dk.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dk
		}
	}

	return nil
}
