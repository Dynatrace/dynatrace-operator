// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScraper(t *testing.T) {
	spec := &ScraperSpec{}

	scraper := NewScraper(spec, "pm")

	assert.Same(t, spec, scraper.ScraperSpec)
	assert.Equal(t, "pm"+ScraperNameSuffix, scraper.GetDeploymentName())
}

func TestScraper_GetDeploymentName(t *testing.T) {
	scraper := NewScraper(&ScraperSpec{}, "pm")

	assert.Equal(t, "pm-scraper", scraper.GetDeploymentName())
}
