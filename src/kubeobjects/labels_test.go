package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructors(t *testing.T) {
	appLabels := NewAppLabels(testComponent, testName, testComponentFeature, testComponentVersion)
	coreLabels := NewCoreLabels(testName, testComponent)

	expectedCoreMatchLabels := map[string]string{
		AppNameLabel:      testAppName,
		AppCreatedByLabel: testName,
		AppComponentLabel: testComponent,
	}
	expectedAppMatchLabels := map[string]string{
		AppNameLabel:      testComponent,
		AppCreatedByLabel: testName,
		AppManagedByLabel: testAppName,
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
	}

	t.Run("verify matchLabels for core", func(t *testing.T) {
		assert.Equal(t, expectedCoreMatchLabels, coreLabels.BuildMatchLabels())
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
