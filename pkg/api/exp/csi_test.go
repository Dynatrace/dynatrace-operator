package exp

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCSIVolumeReadOnly(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   bool
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   false,
		},
		{
			title: "overrule",
			in:    "true",
			out:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				CSIReadOnlyVolumeKey: c.in,
			}}

			out := ff.IsCSIVolumeReadOnly()

			assert.Equal(t, c.out, out)
		})
	}
}

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

func TestGetCSIMaxFailedMountAttempts(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   int
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   DefaultCSIMaxFailedMountAttempts,
		},
		{
			title: "with incorrect value type, negative int",
			in:    "-1",
			out:   DefaultCSIMaxFailedMountAttempts,
		},
		{
			title: "with incorrect value type, too big int",
			in:    "99999999999999999999999999999999999999999999999999999999999999999999999",
			out:   DefaultCSIMaxFailedMountAttempts,
		},
		{
			title: "with incorrect value type, string",
			in:    "not a number",
			out:   DefaultCSIMaxFailedMountAttempts,
		},
		{
			title: "overrule",
			in:    "11",
			out:   11,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				CSIMaxFailedMountAttemptsKey: c.in,
			}}

			out := ff.GetCSIMaxFailedMountAttempts()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestMountAttemptsToTimeout(t *testing.T) {
	type testCase struct {
		in    int
		out   string
	}

	cases := []testCase{
		{
			in:    -100,
			out:   "0s",
		},
		{
			in:    0,
			out:   "0s",
		},
		{
			in:    5,
			out:   "16s",
		},
		{
			in:    8,
			out:   "2m8s",
		},
		{
			in:    10,
			out:   "8m32s",
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("with %d", c.in), func(t *testing.T) {
			out := MountAttemptsToTimeout(c.in)

			assert.Equal(t, c.out, out)
		})
	}
}
