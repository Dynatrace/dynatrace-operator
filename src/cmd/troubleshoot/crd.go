package troubleshoot

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkCRD(troubleshootCtx *troubleshootContext) error {
	log := troubleshootCtx.baseLog.WithName("crd")

	logNewCheckf(log, "checking if CRD for Dynakube exists ...")

	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	err := troubleshootCtx.apiReader.List(troubleshootCtx.context, dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName})

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
