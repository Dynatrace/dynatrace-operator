// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/stretchr/testify/assert"
)

func TestDynaKube_Extensions_IsEnabled(t *testing.T) {
	tests := []struct {
		name             string
		dk               *DynaKube
		databasesEnabled bool
		anyEnabled       bool
	}{
		{
			"empty",
			&DynaKube{},
			false,
			false,
		},
		{
			"empty extensions",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{}}},
			false,
			false,
		},
		{
			"databases enabled",
			&DynaKube{Spec: DynaKubeSpec{Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{{}}}}},
			true,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.databasesEnabled, test.dk.Extensions().IsDatabasesEnabled())
			assert.Equal(t, test.anyEnabled, test.dk.Extensions().IsAnyEnabled())
		})
	}
}
