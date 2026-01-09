package operator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/crdcleanup"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func runCertInit(cfg *rest.Config, namespace string) error {
	return runInitManager(cfg, namespace, certificates.AddInit)
}

func runCRDCleanup(cfg *rest.Config, namespace string) error {
	return runInitManager(cfg, namespace, crdcleanup.AddInit)
}

func runInitManager(cfg *rest.Config, namespace string, addInitFn func(manager.Manager, string, context.CancelFunc) error) error {
	mgr, err := createInitManager(cfg, namespace)
	if err != nil {
		return err
	}

	err = checkCRDs(mgr)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	err = addInitFn(mgr, namespace, cancelFn)
	if err != nil {
		return errors.WithStack(err)
	}

	err = mgr.Start(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func createInitManager(cfg *rest.Config, namespace string) (manager.Manager, error) {
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
