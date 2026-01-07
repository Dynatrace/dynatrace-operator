package operator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/crdcleanup"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func runCRDCleanup(cfg *rest.Config, namespace string) error {
	crdCleanupManager, err := createCRDCleanupManager(cfg, namespace)
	if err != nil {
		return err
	}

	err = checkCRDs(crdCleanupManager)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	err = crdcleanup.AddInit(crdCleanupManager, namespace, cancelFn)
	if err != nil {
		return errors.WithStack(err)
	}

	err = crdCleanupManager.Start(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func createCRDCleanupManager(cfg *rest.Config, namespace string) (manager.Manager, error) {
	controlManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		HealthProbeBindAddress: healthProbeBindAddress,
		LivenessEndpointName:   livenessEndpointName,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = controlManager.AddHealthzCheck(livezEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return controlManager, errors.WithStack(err)
}
