package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
)

func checkOneAgentAPM(ctx *troubleshootContext) error {
	log = newTroubleshootLogger("oneAgentAPM")

	logNewCheckf("checking if OneAgentAPM object exists ...")

	exists, err := kubeobjects.CheckIfOneAgentAPMExists(ctx.apiReader)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("OneAgentAPM object still exists - either delete OneAgentAPM objects or fully install the oneAgent operator")
	}

	logOkf("OneAgentAPM object does not exist")
	return nil
}
