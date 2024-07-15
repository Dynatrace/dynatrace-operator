package token

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
)

func TestFeature_IsScopeMissing(t *testing.T) {
	t.Run("no scope missing", func(t *testing.T) {
		feature := Feature{
			Name:           "Access problem and event feed, metrics, and topology",
			RequiredScopes: []string{"scope1", "scope2"},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		}
		missing, scopes := feature.IsScopeMissing([]string{"scope1", "scope2"})
		assert.False(t, missing)
		assert.Empty(t, scopes)
	})

	t.Run("one scope missing", func(t *testing.T) {
		feature := Feature{
			Name:           "Access problem and event feed, metrics, and topology",
			RequiredScopes: []string{"scope1", "scope2"},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		}
		missing, scopes := feature.IsScopeMissing([]string{"scope1"})
		assert.True(t, missing)
		assert.Equal(t, []string{"scope2"}, scopes)
	})

	t.Run("all scopes missing", func(t *testing.T) {
		feature := Feature{
			Name:           "Access problem and event feed, metrics, and topology",
			RequiredScopes: []string{"scope1", "scope2"},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		}
		missing, scopes := feature.IsScopeMissing([]string{})
		assert.True(t, missing)
		assert.Equal(t, []string{"scope1", "scope2"}, scopes)
	})
}
