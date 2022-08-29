package daemonset

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func DeleteIfExists(name string, namespace string) env.Func {
	return func(ctx context.Context, environmentConfig *envconf.Config) (context.Context, error) {
		var daemonset v1.DaemonSet
		resources := environmentConfig.Client().Resources()

		err := resources.Get(ctx, name, namespace, &daemonset)

		if err != nil && !k8serrors.IsNotFound(err) {
			return ctx, errors.WithStack(err)
		} else if k8serrors.IsNotFound(err) {
			return ctx, nil
		}

		err = resources.Delete(ctx, &daemonset)

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&daemonset))
		return ctx, errors.WithStack(err)
	}
}
