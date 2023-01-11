package support_archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const namespace = "dynatrace"

func TestObjectQuerySyntax(t *testing.T) {
	queries := getQueries(namespace)
	assert.Len(t, queries, 9)

	for _, query := range queries {
		assert.NotEmpty(t, query.groupVersionKind.Kind)
		assert.NotEmpty(t, query.groupVersionKind.Version)
	}
}
