// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScraper(t *testing.T) {
	spec := &ScraperSpec{}

	scraper := NewScraper(spec, "dtp")

	assert.Same(t, spec, scraper.ScraperSpec)
	assert.Equal(t, "dtp"+ScraperNameSuffix, scraper.GetDeploymentName())
}

func TestScraper_GetDeploymentName(t *testing.T) {
	scraper := NewScraper(&ScraperSpec{}, "dtp")

	assert.Equal(t, "dtp-scraper", scraper.GetDeploymentName())
}
