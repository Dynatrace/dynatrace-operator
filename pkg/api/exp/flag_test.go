package exp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetNoProxy(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   string
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   "",
		},
		{
			title: "overrule",
			in:    "my.infra.ca,my.infra.io",
			out:   "my.infra.ca,my.infra.io",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				NoProxyKey: c.in,
			}}

			out := ff.GetNoProxy()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestGetApiRequestThreshold(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   time.Duration
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   DefaultMinRequestThresholdMinutes * time.Minute,
		},
		{
			title: "with incorrect value type, negative int",
			in:    "-1",
			out:   DefaultMinRequestThresholdMinutes * time.Minute,
		},
		{
			title: "with incorrect value type, too big int",
			in:    "99999999999999999999999999999999999999999999999999999999999999999999999",
			out:   DefaultMinRequestThresholdMinutes * time.Minute,
		},
		{
			title: "with incorrect value type, string",
			in:    "not a number",
			out:   DefaultMinRequestThresholdMinutes * time.Minute,
		},
		{
			title: "with incorrect value type, go time format",
			in:    "1h",
			out:   DefaultMinRequestThresholdMinutes * time.Minute,
		},
		{
			title: "overrule",
			in:    "5",
			out:   5 * time.Minute,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				ApiRequestThresholdKey: c.in,
			}}

			out := ff.GetApiRequestThreshold()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestGetIntWithDefault(t *testing.T) {
	type testCase struct {
		title       string
		annotations map[string]string
		targetKey   string
		defaultVal  int
		expected    int
	}

	cases := []testCase{
		{
			title:       "with nil map",
			annotations: nil,
			targetKey:   "",
			defaultVal:  1,
			expected:    1,
		},
		{
			title:       "with empty map",
			annotations: map[string]string{},
			targetKey:   "",
			defaultVal:  2,
			expected:    2,
		},
		{
			title: "with incorrect value type, not int",
			annotations: map[string]string{
				"key": "not-a-number",
			},
			targetKey:  "key",
			defaultVal: 3,
			expected:   3,
		},
		{
			title: "with incorrect value type, too big int",
			annotations: map[string]string{
				"key": "99999999999999999999999999999999999999999999999999999999999999999999999",
			},
			targetKey:  "key",
			defaultVal: 4,
			expected:   4,
		},
		{
			title: "overrule",
			annotations: map[string]string{
				"key": "5",
			},
			targetKey:  "key",
			defaultVal: -1,
			expected:   5,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: c.annotations}

			out := ff.getIntWithDefault(c.targetKey, c.defaultVal)

			assert.Equal(t, c.expected, out)
		})
	}
}

func TestGetDisableFlagWithDeprecatedAnnotation(t *testing.T) {
	type testCase struct {
		title       string
		annotations map[string]string
		targetKey   string
		depKey      string
		expected    bool
	}

	cases := []testCase{
		{
			title:       "with nil map",
			annotations: nil,
			expected:    false,
		},
		{
			title:       "with empty map",
			annotations: map[string]string{},
			expected:    false,
		},
		{
			title: "with incorrect value type, not bool",
			annotations: map[string]string{
				"key": "not-a-bool",
			},
			targetKey: "key",
			expected:  false,
		},
		{
			title: "overrule, non disable key",
			annotations: map[string]string{
				"key": "false",
			},
			targetKey: "key",
			expected:  true,
		},
		{
			title: "overrule, deprecated disable key",
			annotations: map[string]string{
				"key": "false",
			},
			depKey:   "key",
			expected: false,
		},
		{
			title: "overrule, overrule ",
			annotations: map[string]string{
				"disable-f": "false",
				"enable-f":  "false",
			},
			targetKey: "enable-f",
			depKey:    "disable-f",
			expected:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: c.annotations}

			out := ff.getDisableFlagWithDeprecatedAnnotation(c.targetKey, c.depKey)

			assert.Equal(t, c.expected, out)
		})
	}
}

func TestGetFlagBoolWithDefault(t *testing.T) {
	type testCase struct {
		title       string
		annotations map[string]string
		targetKey   string
		defaultVal  bool
		expected    bool
	}

	cases := []testCase{
		{
			title:       "with nil map",
			annotations: nil,
			targetKey:   "doesn't matter",
			defaultVal:  true,
			expected:    true,
		},
		{
			title:       "with empty map",
			annotations: map[string]string{},
			targetKey:   "doesn't matter",
			defaultVal:  true,
			expected:    true,
		},
		{
			title: "with incorrect value type, not bool",
			annotations: map[string]string{
				"key": "not-a-bool",
			},
			targetKey:  "key",
			defaultVal: true,
			expected:   true,
		},
		{
			title: "overrule, with full phrase",
			annotations: map[string]string{
				"key": "false",
			},
			targetKey:  "key",
			defaultVal: true,
			expected:   false,
		},
		{
			title: "overrule, with short phrase",
			annotations: map[string]string{
				"key": "t",
			},
			targetKey:  "key",
			defaultVal: false,
			expected:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: c.annotations}

			out := ff.getBoolWithDefault(c.targetKey, c.defaultVal)

			assert.Equal(t, c.expected, out)
		})
	}
}
