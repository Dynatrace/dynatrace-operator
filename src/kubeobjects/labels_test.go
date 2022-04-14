package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testAppName          = "dynatrace-operator"
	testAppVersion       = "snapshot"
	testName             = "test-name"
	testComponent        = "test-component"
	testComponentFeature = "test-component-feature"
	testComponentVersion = "test-component-version"
)

func TestConstructors(t *testing.T) {
	matchLabels := NewMatchLabels(testName, testComponent)
	podLabels := NewPodLabels(testName, testComponent)
	componentLabels := NewComponentLabels(testName, testComponent, testComponentFeature, testComponentVersion)

	expectedMatchLabels := map[string]string{
		AppNameLabel:      testAppName,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponent,
	}
	expectedPodLabels := map[string]string{
		AppNameLabel:      testAppName,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponent,
		AppVersionLabel:   testAppVersion,
	}
	expectedComponentLabels := map[string]string{
		AppNameLabel:          testAppName,
		AppCreatedByLabel:     testName,
		AppComponentLabel:     testComponent,
		AppVersionLabel:       testAppVersion,
		ComponentFeatureLabel: testComponentFeature,
		ComponentVersionLabel: testComponentVersion,
	}

	t.Run("verify matchLabels", func(t *testing.T) {
		assert.Equal(t, expectedMatchLabels, matchLabels.BuildMatchLabels())
	})

	t.Run("verify matchLabels for pod", func(t *testing.T) {
		assert.Equal(t, expectedMatchLabels, podLabels.BuildMatchLabels())
	})
	t.Run("verify labels for pod", func(t *testing.T) {
		assert.Equal(t, expectedPodLabels, podLabels.BuildLabels())
	})

	t.Run("verify matchLabels for component", func(t *testing.T) {
		assert.Equal(t, expectedMatchLabels, componentLabels.BuildMatchLabels())
	})
	t.Run("verify labels for component pod", func(t *testing.T) {
		assert.Equal(t, expectedComponentLabels, componentLabels.BuildLabels())
	})
}
