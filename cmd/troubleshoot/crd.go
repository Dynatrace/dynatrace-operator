package troubleshoot

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

func checkCRD(baseLog logr.Logger, err error) error {
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
