// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8slabel

import (
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	testAppName    = "dynatrace-operator"
	testAppVersion = "snapshot"
	testName       = "test-name"
)

func TestLabels(t *testing.T) {
	const (
		labelsName     = "labels-test-app"
		labelsInstance = "labels-test-instance"
		labelsVersion  = "labels-test-version"
	)

	tests := []struct {
		name            string
		appVersion      string
		operatorVersion *string
		expectedLabels  map[string]string
		expectedMatch   map[string]string
	}{
		{
			name:       "all labels",
			appVersion: labelsVersion,
			expectedLabels: map[string]string{
				AppName:         labelsName,
				AppInstance:     labelsInstance,
				AppManagedBy:    version.AppName,
				AppVersion:      labelsVersion,
				OperatorVersion: version.Version,
			},
			expectedMatch: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
			},
		},
		{
			name:       "empty workload version",
			appVersion: "",
			expectedLabels: map[string]string{
				AppName:         labelsName,
				AppInstance:     labelsInstance,
				AppManagedBy:    version.AppName,
				OperatorVersion: version.Version,
			},
			expectedMatch: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
			},
		},
		{
			name:            "empty operator version",
			appVersion:      labelsVersion,
			operatorVersion: new(""),
			expectedLabels: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
				AppVersion:   labelsVersion,
			},
			expectedMatch: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
			},
		},
		{
			name:            "long versions",
			appVersion:      strings.Repeat("a", 64),
			operatorVersion: new(strings.Repeat("b", 64)),
			expectedLabels: map[string]string{
				AppName:         labelsName,
				AppInstance:     labelsInstance,
				AppManagedBy:    version.AppName,
				AppVersion:      strings.Repeat("a", 63),
				OperatorVersion: strings.Repeat("b", 63),
			},
			expectedMatch: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
			},
		},
		{
			name:       "app version with invalid character",
			appVersion: "_debug",
			expectedLabels: map[string]string{
				AppName:         labelsName,
				AppInstance:     labelsInstance,
				AppManagedBy:    version.AppName,
				OperatorVersion: version.Version,
			},
			expectedMatch: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
			},
		},
		{
			name:       "app version invalid after truncation",
			appVersion: strings.Repeat("a", 62) + "-suffix",
			expectedLabels: map[string]string{
				AppName:         labelsName,
				AppInstance:     labelsInstance,
				AppManagedBy:    version.AppName,
				OperatorVersion: version.Version,
			},
			expectedMatch: map[string]string{
				AppName:      labelsName,
				AppInstance:  labelsInstance,
				AppManagedBy: version.AppName,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.operatorVersion != nil {
				oldVersion := version.Version
				t.Cleanup(func() {
					version.Version = oldVersion
				})
				version.Version = *tt.operatorVersion
			}

			labels := New(labelsName, labelsInstance, tt.appVersion)

			assert.Equal(t, tt.expectedLabels, labels.AsMap())
			assert.Equal(t, tt.expectedMatch, labels.AsSelector())
		})
	}
}

func TestMergeInto(t *testing.T) {
	labels := New(testAppName, testName, testAppVersion)

	t.Run("sets labels when object has none", func(t *testing.T) {
		cm := &corev1.ConfigMap{}

		labels.MergeInto(cm)

		assert.Equal(t, labels.AsMap(), cm.Labels)
	})
	t.Run("keeps unrelated labels already on the object", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Labels: map[string]string{"keep": "me"}}

		labels.MergeInto(cm)

		assert.Equal(t, "me", cm.Labels["keep"])
		assert.Equal(t, testAppName, cm.Labels[AppName])
	})
	t.Run("overwrites colliding keys", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Labels: map[string]string{AppName: "stale"}}

		labels.MergeInto(cm)

		assert.Equal(t, testAppName, cm.Labels[AppName])
	})
}
