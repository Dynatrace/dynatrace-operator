package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableAttributesDtKubernetes(t *testing.T) {
	type testCase struct {
		title            string
		hasPlatformToken bool
		in               string
		out              bool
	}

	cases := []testCase{
		// Classic token (apiToken) — opt-out: enabled by default, user must explicitly disable.
		{
			title:            "classic token: default (no annotation) => true",
			hasPlatformToken: false,
			in:               "",
			out:              true,
		},
		{
			title:            "classic token: overrule to true",
			hasPlatformToken: false,
			in:               "true",
			out:              true,
		},
		{
			title:            "classic token: overrule to false",
			hasPlatformToken: false,
			in:               "false",
			out:              false,
		},
		// Platform token — opt-in: disabled by default, user must explicitly enable.
		{
			title:            "platform token: default (no annotation) => false",
			hasPlatformToken: true,
			in:               "",
			out:              false,
		},
		{
			title:            "platform token: overrule to true",
			hasPlatformToken: true,
			in:               "true",
			out:              true,
		},
		{
			title:            "platform token: overrule to false",
			hasPlatformToken: true,
			in:               "false",
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
