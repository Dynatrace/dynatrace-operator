package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		client := NewClient(core.NewClient(core.Config{}))
		assert.Implements(t, (*APIClient)(nil), client)
	})
}
