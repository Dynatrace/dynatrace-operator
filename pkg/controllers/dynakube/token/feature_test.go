package token

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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
		scopes := feature.CollectMissingRequiredScopes([]string{"scope1", "scope2"})
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
		scopes := feature.CollectMissingRequiredScopes([]string{"scope1"})
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
		scopes := feature.CollectMissingRequiredScopes([]string{})
		assert.Equal(t, []string{"scope1", "scope2"}, scopes)
	})

	t.Run("no optional scope missing", func(t *testing.T) {
		feature := Feature{
			Name:           "Access problem and event feed, metrics, and topology",
			OptionalScopes: []string{"scope1", "scope2"},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		}
		scopes := feature.CollectMissingOptionalScopes([]string{"scope1", "scope2"})
		assert.Empty(t, scopes)
	})

	t.Run("one optional scope missing", func(t *testing.T) {
		feature := Feature{
			Name:           "Access problem and event feed, metrics, and topology",
			OptionalScopes: []string{"scope1", "scope2"},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		}
		scopes := feature.CollectMissingOptionalScopes([]string{"scope1"})
		assert.Equal(t, []string{"scope2"}, scopes)
	})

	t.Run("all optional scopes missing", func(t *testing.T) {
		feature := Feature{
			Name:           "Access problem and event feed, metrics, and topology",
			OptionalScopes: []string{"scope1", "scope2"},
			IsEnabled: func(dk dynakube.DynaKube) bool {
				return true
			},
		}
		scopes := feature.CollectMissingOptionalScopes([]string{})
		assert.Equal(t, []string{"scope1", "scope2"}, scopes)
	})
}
