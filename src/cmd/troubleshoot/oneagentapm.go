package troubleshoot

import (
	"errors"
	"github.com/Dynatrace/dynatrace-operator/src/util/kubeobjects"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
)

func checkOneAgentAPM(baseLog logr.Logger, kubeConfig *rest.Config) error {
	log := baseLog.WithName("oneAgentAPM")

	logNewCheckf(log, "checking if OneAgentAPM object exists ...")
	exists, err := kubeobjects.CheckIfOneAgentAPMExists(kubeConfig)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("OneAgentAPM object still exists - either delete OneAgentAPM objects or fully install the oneAgent operator")
	}

	logOkf(log, "OneAgentAPM does not exist")
	return nil
}
