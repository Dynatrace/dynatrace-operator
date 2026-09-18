// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildResourceAttributeStatements(t *testing.T) {
	t.Run("empty attrs", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{})
		assert.Empty(t, statements)
	})

	t.Run("single entry", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{"region": "us-east"})
		require.Len(t, statements, 1)
		assert.Equal(t,
			`set(attributes["region"], "us-east") where attributes["region"] == nil`,
			statements[0],
		)
	})

	t.Run("multiple entries sorted by key", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{
			"region":  "us-east",
			"dt.env":  "prod",
			"cluster": "main",
		})
		require.Len(t, statements, 3)
		assert.Equal(t, `set(attributes["cluster"], "main") where attributes["cluster"] == nil`, statements[0])
		assert.Equal(t, `set(attributes["dt.env"], "prod") where attributes["dt.env"] == nil`, statements[1])
		assert.Equal(t, `set(attributes["region"], "us-east") where attributes["region"] == nil`, statements[2])
	})

	t.Run("quotes in key and value are escaped", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{`k"ey`: `val"ue`})
		require.Len(t, statements, 1)
		assert.Equal(t,
			`set(attributes["k\"ey"], "val\"ue") where attributes["k\"ey"] == nil`,
			statements[0],
		)
	})

	t.Run("backslashes in key and value are escaped", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{`k\ey`: `val\ue`})
		require.Len(t, statements, 1)
		assert.Equal(t,
			`set(attributes["k\\ey"], "val\\ue") where attributes["k\\ey"] == nil`,
			statements[0],
		)
	})
}
