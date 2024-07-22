//go:build e2e

package helpers

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube" //nolint:staticcheck
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube" //nolint:staticcheck
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func SetScheme(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
	err := v1beta3.AddToScheme(envConfig.Client().Resources().GetScheme())
	if err != nil {
		return ctx, err
	}
	err = v1beta2.AddToScheme(envConfig.Client().Resources().GetScheme())
	if err != nil {
		return ctx, err
	}
	err = v1beta1.AddToScheme(envConfig.Client().Resources().GetScheme())
	if err != nil {
		return ctx, err
	}
	err = v1alpha1.AddToScheme(envConfig.Client().Resources().GetScheme())
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}
