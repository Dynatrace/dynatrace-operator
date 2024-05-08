package testing

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tHelper interface {
	Helper()
}

// PartialEqual asserts that two objects are equal, depending on what equal means
//
// For instance, you may pass options to ignore certain fields
func PartialEqual(t require.TestingT, expected, actual any, diffOpts cmp.Option, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	if cmp.Equal(expected, actual, diffOpts) {
		return
	}

	diff := cmp.Diff(expected, actual, diffOpts)
	assert.Fail(t, fmt.Sprintf("Not equal: \n"+
		"expected: %s\n"+
		"actual  : %s%s", expected, actual, diff), msgAndArgs...)
}
