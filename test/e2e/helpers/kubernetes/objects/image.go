//go:build e2e

package objects

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func VerifyWorkloadUsesImage[PT k8s.Object](obj PT, name, namespace, expectedImage string, getContainers func(PT) []corev1.Container) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Get(ctx, name, namespace, obj))

		for _, c := range getContainers(obj) {
			if c.Image == expectedImage {
				return ctx
			}
		}

		assert.Failf(t, "image not used", "expected image %q not found in %T %q containers", expectedImage, obj, name)

		return ctx
	}
}
