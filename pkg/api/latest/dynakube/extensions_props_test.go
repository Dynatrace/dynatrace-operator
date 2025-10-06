package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/stretchr/testify/require"
)

func TestDynaKube_Extensions_IsEnabled(t *testing.T) {
	tests := []struct {
		name          string
		dk            *DynaKube
		expectEnabled bool
	}{
		{
			"empty",
			&DynaKube{},
			false,
		},
		{
			"empty extensions",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{}}},
			false,
		},
		{
			"prometheus enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}},
			true,
		},
		{
			"databases enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{}}}}},
			true,
		},
		{
			"both enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}, Databases: []extensions.DatabaseSpec{{}}}}},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectEnabled, test.dk.Extensions().IsEnabled())
		})
	}
}
