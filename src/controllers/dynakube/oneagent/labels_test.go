package oneagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildLabels(t *testing.T) {
	l := buildLabels("my-name", "classic")
	assert.Equal(t, l["dynatrace.com/component"], "operator")
	assert.Equal(t, l["operator.dynatrace.com/instance"], "my-name")
	assert.Equal(t, l["operator.dynatrace.com/feature"], "classic")
}
