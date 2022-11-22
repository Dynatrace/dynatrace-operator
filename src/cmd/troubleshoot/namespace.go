package troubleshoot

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespace(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("namespace")

	logNewCheckf("checking if namespace '%s' exists ...", troubleshootCtx.namespaceName)

	var namespace corev1.Namespace
	err := troubleshootCtx.apiReader.Get(troubleshootCtx.context, client.ObjectKey{Name: troubleshootCtx.namespaceName}, &namespace)

	if err != nil {
		return errors.Wrapf(err, "missing namespace '%s'", troubleshootCtx.namespaceName)
	}

	logOkf("using namespace '%s'", troubleshootCtx.namespaceName)
	return nil
}
