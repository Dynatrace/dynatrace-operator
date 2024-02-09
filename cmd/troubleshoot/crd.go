package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

func checkCRD(baseLog logger.DtLogger, err error) error {
	log := baseLog.WithName("crd")

	logNewCheckf(log, "checking if CRD for Dynakube exists ...")

	if err != nil {
		return DetermineDynakubeError(err)
	}

	logOkf(log, "CRD for Dynakube exists")

	return nil
}

func DetermineDynakubeError(err error) error {
	if runtime.IsNotRegisteredError(err) {
		err = errors.Wrap(err, "CRD for Dynakube missing")
	} else {
		err = errors.Wrap(err, "could not list Dynakube")
	}

	return err
}
