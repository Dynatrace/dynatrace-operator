// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScraper(t *testing.T) {
	spec := &ScraperSpec{}

	scraper := NewScraper(spec, "dtprom")

	assert.Same(t, spec, scraper.ScraperSpec)
	assert.Equal(t, "dtprom"+ScraperNameSuffix, scraper.GetDeploymentName())
}

func TestScraper_GetDeploymentName(t *testing.T) {
	scraper := NewScraper(&ScraperSpec{}, "dtprom")

	assert.Equal(t, "dtprom-scraper", scraper.GetDeploymentName())
}
