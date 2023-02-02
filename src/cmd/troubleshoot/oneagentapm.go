package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkOneAgentAPM(clt client.Client, ctx *troubleshootContext) error {
	log = newTroubleshootLogger("oneAgentAPM")

	logNewCheckf("checking if OneAgentAPM object exists ...")

	exists, err := kubeobjects.CheckIfOneAgentAPMExists(clt)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("OneAgentAPM object still exists")
	}

	logOkf("OneAgentAPM object does not exist")
	return nil
}
