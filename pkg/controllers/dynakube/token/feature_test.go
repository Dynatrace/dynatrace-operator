// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeature_CollectMissingRequiredScopes(t *testing.T) {
	type testCase struct {
		title           string
		requiredScopes  []string
		availableScopes []string
		expectedMissing []string
	}

	cases := []testCase{
		{
			title:           "no scope missing",
			requiredScopes:  []string{"scope1", "scope2"},
			availableScopes: []string{"scope1", "scope2"},
			expectedMissing: []string{},
		},
		{
			title:           "one scope missing",
			requiredScopes:  []string{"scope1", "scope2"},
			availableScopes: []string{"scope2"},
			expectedMissing: []string{"scope1"},
		},
		{
			title:           "all scopes missing",
			requiredScopes:  []string{"scope1", "scope2"},
			availableScopes: []string{},
			expectedMissing: []string{"scope1", "scope2"},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			feature := Feature{
				RequiredScopes: c.requiredScopes,
			}
			missingScopes := feature.CollectMissingRequiredScopes(c.availableScopes)
			assert.Equal(t, c.expectedMissing, missingScopes)
		})
	}
}
