package troubleshoot

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespace(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string) error {
	log := baseLog.WithName("namespace")

	logNewCheckf(log, "checking if namespace '%s' exists ...", namespaceName)

	var namespace corev1.Namespace

	err := apiReader.Get(ctx, client.ObjectKey{Name: namespaceName}, &namespace)
	if err != nil {
		return errors.Wrapf(err, "missing namespace '%s'", namespaceName)
	}

	logOkf(log, "using namespace '%s'", namespaceName)

	return nil
}
