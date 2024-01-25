package support_archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const namespace = "dynatrace"

func TestObjectQuerySyntax(t *testing.T) {
	queries := getQueries(namespace, defaultOperatorAppName)
	assert.Len(t, queries, 17)

	for _, query := range queries {
		assert.NotEmpty(t, query.groupVersionKind.Kind)
		assert.NotEmpty(t, query.groupVersionKind.Version)
	}
}
