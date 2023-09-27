package troubleshoot

import (
	"context"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkCRD(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string) error {
	log := baseLog.WithName("crd")

	logNewCheckf(log, "checking if CRD for Dynakube exists ...")

	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	err := apiReader.List(ctx, dynakubeList, &client.ListOptions{Namespace: namespaceName})

	if err != nil {
		return determineDynakubeError(err)
	}

	logOkf(log, "CRD for Dynakube exists")
	return nil
}

func determineDynakubeError(err error) error {
	if runtime.IsNotRegisteredError(err) {
		err = errors.Wrap(err, "CRD for Dynakube missing")
	} else {
		err = errors.Wrap(err, "could not list Dynakube")
	}
	return err
}
