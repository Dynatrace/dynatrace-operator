package nodes

import (
	"context"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileNodes) determineDynakubeForNode(nodeName string) (*dynatracev1beta1.DynaKube, error) {
	dkList, err := r.getOneAgentList()
	if err != nil {
		return nil, err
	}

	return r.filterOneAgentFromList(dkList, nodeName), nil
}

func (r *ReconcileNodes) getOneAgentList() (*dynatracev1beta1.DynaKubeList, error) {
	watchNamespace := os.Getenv("POD_NAMESPACE")

	var dkList dynatracev1beta1.DynaKubeList
	err := r.client.List(context.TODO(), &dkList, client.InNamespace(watchNamespace))
	if err != nil {
		return nil, err
	}

	return &dkList, nil
}

func (r *ReconcileNodes) filterOneAgentFromList(dkList *dynatracev1beta1.DynaKubeList,
	nodeName string) *dynatracev1beta1.DynaKube {

	for _, dynakube := range dkList.Items {
		items := dynakube.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dynakube
		}
	}

	return nil
}
