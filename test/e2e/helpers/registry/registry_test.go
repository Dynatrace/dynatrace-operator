// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_selectTag(t *testing.T) {
	tags := []string{
		"sha256-e6706016aa4d7bc6efe98a9cbd3077f97c74416e23d7967037847b828303d3cf.sig",
		"1.331.64.20260903-230837-fips",
		"1.331.64.20260903-230837",
		"sha256-752b19493f392f05e9e46feff42c92a534c828bfe7897551c88c56f246d057ca.att",
		"1.315.25.20250527-232755-fips",
		"1.337.28.20260501-093315-java21-fips",
		"1.329-fips",
		"latest",
		"1.327.30.20251107-111521-python",
		"1.329.91.20260908-125356",
		"1.329.67.20260112-133153-java",
	}

	t.Run("no matches with FIPS", func(t *testing.T) {
		got := selectTag([]string{
			"sha256-e6706016aa4d7bc6efe98a9cbd3077f97c74416e23d7967037847b828303d3cf.sig",
			"1.331.64.20260903-230837",
			"sha256-752b19493f392f05e9e46feff42c92a534c828bfe7897551c88c56f246d057ca.att",
			"1.337.28.20260501-093315-java21-fips",
			"1.329-fips",
			"latest",
			"1.327.30.20251107-111521-python",
			"1.329.91.20260908-125356",
			"1.329.67.20260112-133153-java",
		}, 0, true)
		assert.Empty(t, got)
	})

	t.Run("no matches without FIPS", func(t *testing.T) {
		got := selectTag([]string{
			"sha256-e6706016aa4d7bc6efe98a9cbd3077f97c74416e23d7967037847b828303d3cf.sig",
			"1.331.64.20260903-230837-fips",
			"sha256-752b19493f392f05e9e46feff42c92a534c828bfe7897551c88c56f246d057ca.att",
			"1.315.25.20250527-232755-fips",
			"1.337.28.20260501-093315-java21-fips",
			"1.329-fips",
			"latest",
			"1.327.30.20251107-111521-python",
			"1.329.67.20260112-133153-java",
		}, 0, false)
		assert.Empty(t, got)
	})

	t.Run("with FIPS", func(t *testing.T) {
		got := selectTag(tags, 0, true)
		assert.Equal(t, "1.331.64.20260903-230837-fips", got)
	})

	t.Run("without FIPS", func(t *testing.T) {
		got := selectTag(tags, 0, false)
		assert.Equal(t, "1.331.64.20260903-230837", got)
	})

	t.Run("previous with FIPS", func(t *testing.T) {
		got := selectTag(tags, 1, true)
		assert.Equal(t, "1.315.25.20250527-232755-fips", got)
	})

	t.Run("previous without FIPS", func(t *testing.T) {
		got := selectTag(tags, 1, false)
		assert.Equal(t, "1.329.91.20260908-125356", got)
	})

	t.Run("offset beyond available tags returns empty", func(t *testing.T) {
		got := selectTag([]string{"1.329.91.20260908-125356"}, 1, false)
		assert.Empty(t, got)
	})
}
