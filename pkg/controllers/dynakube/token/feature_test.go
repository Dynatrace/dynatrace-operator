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

func TestFeature_CollectOptionalScopes(t *testing.T) {
	type testCase struct {
		title           string
		optionalScopes  []string
		availableScopes []string
		expectedOut     map[string]bool
	}

	cases := []testCase{
		{
			title:           "no scope missing",
			optionalScopes:  []string{"scope1", "scope2"},
			availableScopes: []string{"scope1", "scope2"},
			expectedOut: map[string]bool{
				"scope1": true,
				"scope2": true,
			},
		},
		{
			title:           "one scope missing",
			optionalScopes:  []string{"scope1", "scope2"},
			availableScopes: []string{"scope2"},
			expectedOut: map[string]bool{
				"scope1": false,
				"scope2": true,
			},
		},
		{
			title:           "all scopes missing",
			optionalScopes:  []string{"scope1", "scope2"},
			availableScopes: []string{},
			expectedOut: map[string]bool{
				"scope1": false,
				"scope2": false,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			feature := Feature{
				OptionalScopes: c.optionalScopes,
			}
			optionalScopes := feature.CollectOptionalScopes(c.availableScopes)
			assert.Equal(t, c.expectedOut, optionalScopes)
		})
	}
}
