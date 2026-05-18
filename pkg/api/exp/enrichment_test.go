package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableAttributesDtKubernetes(t *testing.T) {
	type testCase struct {
		title            string
		in               string
		hasPlatformToken bool
		out              bool
	}

	cases := []testCase{
		// Classic token (apiToken) — opt-out: enabled by default, user must explicitly disable.
		{
			title:            "classic token: default (no annotation) => true",
			in:               "",
			hasPlatformToken: false,
			out:              true,
		},
		{
			title:            "classic token: overrule to true",
			in:               "true",
			hasPlatformToken: false,
			out:              true,
		},
		{
			title:            "classic token: overrule to false",
			in:               "false",
			hasPlatformToken: false,
			out:              false,
		},
		{
			title:            "classic token: uppercase TRUE",
			in:               "TRUE",
			hasPlatformToken: false,
			out:              true,
		},
		{
			title:            "classic token: uppercase FALSE",
			in:               "FALSE",
			hasPlatformToken: false,
			out:              false,
		},
		// Platform token — opt-in: disabled by default, user must explicitly enable.
		{
			title:            "platform token: default (no annotation) => false",
			in:               "",
			hasPlatformToken: true,
			out:              false,
		},
		{
			title:            "platform token: overrule to true",
			in:               "true",
			hasPlatformToken: true,
			out:              true,
		},
		{
			title:            "platform token: overrule to false",
			in:               "false",
			hasPlatformToken: true,
			out:              false,
		},
		{
			title:            "platform token: uppercase TRUE",
			in:               "TRUE",
			hasPlatformToken: true,
			out:              true,
		},
		{
			title:            "platform token: uppercase FALSE",
			in:               "FALSE",
			hasPlatformToken: true,
			out:              false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{
				annotations:      map[string]string{EnrichmentEnableAttributesDTKubernetes: c.in},
				hasPlatformToken: c.hasPlatformToken,
			}

			out := ff.EnableAttributesDTKubernetes()

			assert.Equal(t, c.out, out)
		})
	}
}
