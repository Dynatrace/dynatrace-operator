package k8slabel

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
)

const (
	testAppName          = "dynatrace-operator"
	testAppVersion       = "snapshot"
	testName             = "test-name"
	testComponent        = "test-component"
	testComponentFeature = "test-component-feature"
	testComponentVersion = "test-component-version"
	testLongVersion      = "test-0000-test-1111-test-2222-test-3333-test-4444-test-long-5555-test-6666"
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

func TestLongVersion(t *testing.T) {
	appLabels := NewAppLabels(testComponent, testName, testComponentFeature, testLongVersion)

	oldVersion := version.Version
	version.Version = testLongVersion

	coreLabels := NewCoreLabels(testName, testComponent)

	version.Version = oldVersion

	assert.Len(t, appLabels.Version, 63)
	assert.Len(t, coreLabels.Version, 63)
}
