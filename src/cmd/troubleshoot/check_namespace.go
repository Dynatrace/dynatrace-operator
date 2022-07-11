package troubleshoot

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespace(apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.SetPrefix("[namespace ] ")

	tslog.NewTestf("checking if namespace '%s' exists ...", troubleshootContext.namespaceName)

	var namespace corev1.Namespace
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.namespaceName}, &namespace); err != nil {
		tslog.WithErrorf(err, "missing namespace '%s'", troubleshootContext.namespaceName)
		return err
	}

	tslog.Okf("using namespace '%s'", troubleshootContext.namespaceName)
	return nil
}
