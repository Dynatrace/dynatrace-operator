// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package manifests

import (
	"context"
	"encoding/json"
	"os"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func applyHandler(r *resources.Resources) decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		return r.Patch(ctx, obj, k8s.Patch{
			PatchType: types.ApplyPatchType,
			Data:      data,
		}, func(opts *metav1.PatchOptions) {
			opts.FieldManager = "e2e-test"
			opts.Force = new(true)
		})
	}
}

func InstallFromFile(path string, options ...decoder.DecodeOption) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		kubernetesManifest, err := os.Open(path)
		if err != nil {
			return ctx, err
		}
		defer func() { kubernetesManifest.Close() }()

		resources := envConfig.Client().Resources()

		return ctx, decoder.DecodeEach(ctx, kubernetesManifest, applyHandler(resources), options...)
	}
}

func UninstallFromFile(path string, options ...decoder.DecodeOption) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		kubernetesManifest, err := os.Open(path)
		if err != nil {
			return ctx, err
		}
		defer func() { kubernetesManifest.Close() }()

		resources := envConfig.Client().Resources()

		return ctx, decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.DeleteHandler(resources), k8serrors.IsNotFound), options...)
	}
}
