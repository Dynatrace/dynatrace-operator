// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSumErrors(t *testing.T) {
	assert.Equal(t, SumErrors([]string{"asd"}, "1"), "\n1 error(s) found in the 1\n 1. asd")
}
