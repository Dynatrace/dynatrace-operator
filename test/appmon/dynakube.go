package appmon

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func dataIngestDynakube(apiUrl string) *v1beta1.DynaKube {
	instance := dynakube.NewDynakube()
	instance.Spec = v1beta1.DynaKubeSpec{
		APIURL: apiUrl,
		OneAgent: v1beta1.OneAgentSpec{
			ApplicationMonitoring: &v1beta1.ApplicationMonitoringSpec{
				UseCSIDriver: address.Of(false),
			},
		},
	}

	return &instance
}

func applyDynakube(apiUrl string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, dataIngestDynakube(apiUrl)))
		return ctx
	}
}
