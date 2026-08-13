// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8slabel

import (
	"strings"
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

func TestLabels(t *testing.T) {
	const (
		labelsName           = "labels-test-app"
		labelsCreatedBy      = "labels-test-created-by"
		labelsComponent      = "labels-test-component"
		labelsVersion        = "labels-test-version"
		labelsManagerVersion = "labels-test-manager-version"
	)

	tests := []struct {
		name           string
		component      string
		appVersion     string
		managerVersion string
		expectedLabels map[string]string
		expectedMatch  map[string]string
	}{
		{
			name:           "all labels",
			component:      labelsComponent,
			appVersion:     labelsVersion,
			managerVersion: labelsManagerVersion,
			expectedLabels: map[string]string{
				AppNameLabel:           labelsName,
				AppCreatedByLabel:      labelsCreatedBy,
				AppManagedByLabel:      version.AppName,
				AppComponentLabel:      labelsComponent,
				AppVersionLabel:        labelsVersion,
				AppManagerVersionLabel: labelsManagerVersion,
			},
			expectedMatch: map[string]string{
				AppNameLabel:      labelsName,
				AppCreatedByLabel: labelsCreatedBy,
				AppManagedByLabel: version.AppName,
				AppComponentLabel: labelsComponent,
			},
		},
		{
			name:           "empty workload version",
			component:      labelsComponent,
			appVersion:     "",
			managerVersion: labelsManagerVersion,
			expectedLabels: map[string]string{
				AppNameLabel:           labelsName,
				AppCreatedByLabel:      labelsCreatedBy,
				AppManagedByLabel:      version.AppName,
				AppComponentLabel:      labelsComponent,
				AppManagerVersionLabel: labelsManagerVersion,
			},
			expectedMatch: map[string]string{
				AppNameLabel:      labelsName,
				AppCreatedByLabel: labelsCreatedBy,
				AppManagedByLabel: version.AppName,
				AppComponentLabel: labelsComponent,
			},
		},
		{
			name:           "empty manager version",
			component:      labelsComponent,
			appVersion:     labelsVersion,
			managerVersion: "",
			expectedLabels: map[string]string{
				AppNameLabel:      labelsName,
				AppCreatedByLabel: labelsCreatedBy,
				AppManagedByLabel: version.AppName,
				AppComponentLabel: labelsComponent,
				AppVersionLabel:   labelsVersion,
			},
			expectedMatch: map[string]string{
				AppNameLabel:      labelsName,
				AppCreatedByLabel: labelsCreatedBy,
				AppManagedByLabel: version.AppName,
				AppComponentLabel: labelsComponent,
			},
		},
		{
			name:           "long versions",
			component:      labelsComponent,
			appVersion:     strings.Repeat("a", 64),
			managerVersion: strings.Repeat("b", 64),
			expectedLabels: map[string]string{
				AppNameLabel:           labelsName,
				AppCreatedByLabel:      labelsCreatedBy,
				AppManagedByLabel:      version.AppName,
				AppComponentLabel:      labelsComponent,
				AppVersionLabel:        strings.Repeat("a", 63),
				AppManagerVersionLabel: strings.Repeat("b", 63),
			},
			expectedMatch: map[string]string{
				AppNameLabel:      labelsName,
				AppCreatedByLabel: labelsCreatedBy,
				AppManagedByLabel: version.AppName,
				AppComponentLabel: labelsComponent,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion := version.Version
			t.Cleanup(func() {
				version.Version = oldVersion
			})
			version.Version = tt.managerVersion

			labels := New(labelsName, labelsCreatedBy, tt.component, tt.appVersion)

			assert.Equal(t, tt.expectedLabels, labels.Build())
			assert.Equal(t, tt.expectedMatch, labels.BuildMatch())
		})
	}
}
