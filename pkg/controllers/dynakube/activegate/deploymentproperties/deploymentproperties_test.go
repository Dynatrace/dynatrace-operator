// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package deploymentproperties

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildContent(t *testing.T) {
	t.Run("nil map produces empty string", func(t *testing.T) {
		content := BuildContent(nil)
		assert.Empty(t, content)
	})

	t.Run("empty map produces empty string", func(t *testing.T) {
		content := BuildContent(map[string]string{})
		assert.Empty(t, content)
	})

	t.Run("single entry", func(t *testing.T) {
		content := BuildContent(map[string]string{"foo": "bar"})
		assert.Equal(t, "[resource_attributes]\nfoo = bar\n", content)
	})

	t.Run("multiple entries are sorted by key", func(t *testing.T) {
		attrs := map[string]string{
			"zzz": "last",
			"aaa": "first",
			"mmm": "middle",
		}
		content := BuildContent(attrs)
		assert.Equal(t, "[resource_attributes]\naaa = first\nmmm = middle\nzzz = last\n", content)
	})

	t.Run("sanitize input", func(t *testing.T) {
		attrs := map[string]string{
			"foo\nbar": "sa\nnit\riz\ted",
		}
		content := BuildContent(attrs)
		assert.Equal(t, "[resource_attributes]\nfoo_bar = sanitized\n", content)
	})
}
