package namespace

import (
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func Create(name string) env.Func {
	return func(ctx context.Context, environmentConfig *envconf.Config) (context.Context, error) {
		namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
		err := environmentConfig.Client().Resources().Create(ctx, namespace)
		return ctx, errors.WithStack(err)
	}
}

func Delete(name string) env.Func {
	return func(ctx context.Context, environmentConfig *envconf.Config) (context.Context, error) {
		namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
		err := environmentConfig.Client().Resources().Delete(ctx, namespace)
		return ctx, errors.WithStack(err)
	}
}
