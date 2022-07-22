package troubleshoot

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespace(troubleshootCtx *troubleshootContext) error {
	tslog.SetPrefix("[namespace ] ")

	tslog.NewTestf("checking if namespace '%s' exists ...", troubleshootCtx.namespaceName)

	var namespace corev1.Namespace
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.namespaceName}, &namespace); err != nil {
		tslog.WithErrorf(err, "missing namespace '%s'", troubleshootCtx.namespaceName)
		return err
	}

	tslog.Okf("using namespace '%s'", troubleshootCtx.namespaceName)
	return nil
}
