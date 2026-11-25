// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package exp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCSIMaxRetryTimeout(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   time.Duration
	}

	defaultDuration, err := time.ParseDuration(DefaultCSIMaxMountTimeout)
	require.NoError(t, err)

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   defaultDuration,
		},
		{
			title: "with incorrect value type, negative int",
			in:    "-1",
			out:   defaultDuration,
		},
		{
			title: "with incorrect value type, too big int",
			in:    "99999999999999999999999999999999999999999999999999999999999999999999999",
			out:   defaultDuration,
		},
		{
			title: "with incorrect value type, string",
			in:    "not a number",
			out:   defaultDuration,
		},
		{
			title: "with incorrect value type, int",
			in:    "5",
			out:   defaultDuration,
		},
		{
			title: "overrule",
			in:    "1h",
			out:   1 * time.Hour,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				CSIMaxMountTimeoutKey: c.in,
			}}

			out := ff.GetCSIMaxRetryTimeout()

			assert.Equal(t, c.out, out)
		})
	}
}
