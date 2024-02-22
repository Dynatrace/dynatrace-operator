//go:build e2e

package manifests

import (
	"context"
	"os"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func InstallFromFile(path string, options ...decoder.DecodeOption) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		kubernetesManifest, err := os.Open(path)
		defer func() { kubernetesManifest.Close() }()
		if err != nil {
			return ctx, err
		}

		resources := envConfig.Client().Resources()
		err = decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), k8serrors.IsAlreadyExists), options...)
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}

func UninstallFromFile(path string, options ...decoder.DecodeOption) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		kubernetesManifest, err := os.Open(path)
		defer func() { kubernetesManifest.Close() }()
		if err != nil {
			return ctx, err
		}

		resources := envConfig.Client().Resources()
		err = decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.DeleteHandler(resources), k8serrors.IsNotFound), options...)
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}
