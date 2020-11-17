package dynakube

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func GetInstance(clt client.Client, request reconcile.Request) (*dynatracev1alpha1.DynaKube, error) {
	var instance *dynatracev1alpha1.DynaKube
	err := clt.Get(context.TODO(), request.NamespacedName, instance)
	return instance, err
}
