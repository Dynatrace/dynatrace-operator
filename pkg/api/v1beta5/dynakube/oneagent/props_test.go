// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOneAgentDaemonsetName(t *testing.T) {
	oneAgent := OneAgent{name: "test-name"}
	assert.Equal(t, "test-name-oneagent", oneAgent.GetDaemonsetName())
}
