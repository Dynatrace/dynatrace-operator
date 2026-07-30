// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNew(t *testing.T) {
	spec := &Spec{}

	scraper := New(spec, "dtprom")

	assert.Same(t, spec, scraper.Spec)
	assert.Equal(t, "dtprom"+NameSuffix, scraper.GetName())
}

func TestScraper_GetName(t *testing.T) {
	scraper := New(&Spec{}, "dtprom")

	assert.Equal(t, "dtprom-scraper", scraper.GetName())
}

func TestScraper_SetName(t *testing.T) {
	scraper := New(&Spec{}, "old")

	scraper.SetName("new")

	assert.Equal(t, "new-scraper", scraper.GetName())
}

func TestScraper_GetUpdateStrategy(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: new(intstr.FromInt32(1)),
		},
	}

	scraper := New(&Spec{UpdateStrategy: strategy}, "dtprom")

	assert.Equal(t, strategy, scraper.GetUpdateStrategy())
}

// TestScraper_PromotesCommonGetters verifies that the shared common.Spec getters
// are promoted through the wrapper's embedded *Spec.
func TestScraper_PromotesCommonGetters(t *testing.T) {
	scraper := New(&Spec{}, "dtprom")
	assert.Equal(t, int32(2), scraper.GetReplicas())

	scraper.Replicas = new(int32(3))
	assert.Equal(t, int32(3), scraper.GetReplicas())
}
