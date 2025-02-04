//go:build e2e

package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringifier(t *testing.T) {
	assert.Equal(t, "du -c /foobar", DiskUsageWithTotal("/foobar").String())
	assert.Equal(t, "tail -n 1", FilterLastLineOnly().String())
	assert.Equal(t, "du -c /foobar | tail -n 1", Pipe(DiskUsageWithTotal("/foobar"), FilterLastLineOnly()).String())
}
