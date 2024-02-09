package troubleshoot

import (
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oneagentapm"
	"k8s.io/client-go/rest"
)

func checkOneAgentAPM(baseLog logger.DtLogger, kubeConfig *rest.Config) error {
	log := baseLog.WithName("oneAgentAPM")

	logNewCheckf(log, "checking if OneAgentAPM object exists ...")

	exists, err := oneagentapm.Exists(kubeConfig)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("OneAgentAPM object still exists - either delete OneAgentAPM objects or fully install the oneAgent operator")
	}

	logOkf(log, "OneAgentAPM does not exist")

	return nil
}
