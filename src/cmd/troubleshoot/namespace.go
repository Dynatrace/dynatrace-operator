package troubleshoot

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespace(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[namespace ] ")

	logNewTestf("checking if namespace '%s' exists ...", troubleshootCtx.namespaceName)

	var namespace corev1.Namespace
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.namespaceName}, &namespace); err != nil {
		logWithErrorf(err, "missing namespace '%s'", troubleshootCtx.namespaceName)
		return err
	}

	logOkf("using namespace '%s'", troubleshootCtx.namespaceName)
	return nil
}
