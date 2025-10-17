package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/stretchr/testify/assert"
)

func TestDynaKube_Extensions_IsEnabled(t *testing.T) {
	tests := []struct {
		name              string
		dk                *DynaKube
		prometheusEnabled bool
		databasesEnabled  bool
		anyEnabled        bool
	}{
		{
			"empty",
			&DynaKube{},
			false,
			false,
			false,
		},
		{
			"empty extensions",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{}}},
			false,
			false,
			false,
		},
		{
			"prometheus enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}}},
			true,
			false,
			true,
		},
		{
			"databases enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{}}}}},
			false,
			true,
			true,
		},
		{
			"both enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}, Databases: []extensions.DatabaseSpec{{}}}}},
			true,
			true,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.prometheusEnabled, test.dk.Extensions().IsPrometheusEnabled())
			assert.Equal(t, test.databasesEnabled, test.dk.Extensions().IsDatabasesEnabled())
			assert.Equal(t, test.anyEnabled, test.dk.Extensions().IsAnyEnabled())
		})
	}
}
