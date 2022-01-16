package nodes

import (
	"context"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func determineDynakubeForNode(nodeName string, mgrClient client.Client) (*dynatracev1beta1.DynaKube, error) {
	dks, err := getDynakubeList(mgrClient)
	if err != nil {
		return nil, err
	}
	return filterOneAgentFromList(dks, nodeName), nil
}

func getDynakubeList(mgrClient client.Client) (*dynatracev1beta1.DynaKubeList, error) {
	watchNamespace := os.Getenv("POD_NAMESPACE")
	var dynakubeList dynatracev1beta1.DynaKubeList
	err := mgrClient.List(context.TODO(), &dynakubeList, client.InNamespace(watchNamespace))
	if err != nil {
		return nil, err
	}
	return &dynakubeList, nil
}

func filterOneAgentFromList(dkList *dynatracev1beta1.DynaKubeList,
	nodeName string) *dynatracev1beta1.DynaKube {

	for _, dynakube := range dkList.Items {
		items := dynakube.Status.OneAgent.Instances
		if _, ok := items[nodeName]; ok {
			return &dynakube
		}
	}

	return nil
}
