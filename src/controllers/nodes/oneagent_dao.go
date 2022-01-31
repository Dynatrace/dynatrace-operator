package nodes

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileNode) determineDynakubeForNode(nodeName string) (*dynatracev1beta1.DynaKube, error) {
	dks, err := r.getDynakubeList()
	if err != nil {
		return nil, err
	}
	return r.filterOneAgentFromList(dks, nodeName), nil
}

func (r *ReconcileNode) getDynakubeList() (*dynatracev1beta1.DynaKubeList, error) {
	var dynakubeList dynatracev1beta1.DynaKubeList
	err := r.client.List(context.TODO(), &dynakubeList, client.InNamespace(r.podNamespace))
	if err != nil {
		return nil, err
	}
	return &dynakubeList, nil
}

func (r *ReconcileNode) filterOneAgentFromList(dkList *dynatracev1beta1.DynaKubeList,
	nodeName string) *dynatracev1beta1.DynaKube {

	for _, dynakube := range dkList.Items {
		items := dynakube.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dynakube
		}
	}
	return nil
}
