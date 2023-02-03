package troubleshoot

import (
	"github.com/pkg/errors"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkOneAgentAPM(ctx *troubleshootContext) error {
	log = newTroubleshootLogger("oneAgentAPM")

	logNewCheckf("checking if OneAgentAPM object exists ...")

	crdList := &apiv1.CustomResourceDefinitionList{}
	err := ctx.apiReader.List(ctx.context, crdList, &client.ListOptions{})

	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			logOkf("OneAgentAPM does not exist")
			return nil
		}
		return err
	}

	for _, crd := range crdList.Items {
		if crd.Kind == "OneAgentAPM" {
			return errors.Wrap(err, "OneAgentAPM still exists - either delete OneAgentAPM objects or fully install the oneAgent operator")
		}
	}

	logOkf("OneAgentAPM does not exist")
	return nil
}
