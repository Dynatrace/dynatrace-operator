package labels

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

	t.Run("verify matchLabels for statefulsetreconciler", func(t *testing.T) {
		assert.Equal(t, expectedCoreMatchLabels, coreLabels.BuildMatchLabels())
	})
	t.Run("verify labels for statefulsetreconciler", func(t *testing.T) {
		assert.Equal(t, expectedCoreLabels, coreLabels.BuildLabels())
	})

	t.Run("verify matchLabels for app", func(t *testing.T) {
		assert.Equal(t, expectedAppMatchLabels, appLabels.BuildMatchLabels())
	})
	t.Run("verify labels for app", func(t *testing.T) {
		assert.Equal(t, expectedAppLabels, appLabels.BuildLabels())
	})
}
