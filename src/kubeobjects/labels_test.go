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
	matchLabels := newMatchLabels(testAppName, testName, testComponent)
	appLabels := NewAppLabels(testComponent, testName, testComponentVersion, testComponentFeature)
	coreLabels := NewCoreLabels(testName, testComponent)

	expectedMatchLabels := map[string]string{
		AppNameLabel:      testAppName,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponent,
	}
	expectedAppMatchLabels := map[string]string{
		AppNameLabel:      testComponent,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponentFeature,
	}
	expectedAppLabels := map[string]string{
		AppNameLabel:      testComponent,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponentFeature,
		AppVersionLabel:   testComponentVersion,
		AppManagedByLabel: testAppName,
	}
	expectedCoreLabels := map[string]string{
		AppNameLabel:      testAppName,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponent,
		AppVersionLabel:   testAppVersion,
		AppManagedByLabel: testAppName,
	}

	t.Run("verify matchLabels", func(t *testing.T) {
		assert.Equal(t, expectedMatchLabels, matchLabels.BuildMatchLabels())
	})

	t.Run("verify matchLabels for core", func(t *testing.T) {
		assert.Equal(t, expectedMatchLabels, coreLabels.BuildMatchLabels())
	})
	t.Run("verify labels for core", func(t *testing.T) {
		assert.Equal(t, expectedCoreLabels, coreLabels.BuildLabels())
	})

	t.Run("verify matchLabels for app", func(t *testing.T) {
		assert.Equal(t, expectedAppMatchLabels, appLabels.BuildMatchLabels())
	})
	t.Run("verify labels for app", func(t *testing.T) {
		assert.Equal(t, expectedAppLabels, appLabels.BuildLabels())
	})
}
