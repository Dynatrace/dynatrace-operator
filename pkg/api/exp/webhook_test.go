package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableAttributesDtKubernetes(t *testing.T) {
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
			title: "overrule to true",
			in:    "true",
			out:   true,
		},
		{
			title: "overrule to false",
			in:    "false",
			out:   false,
		},
		{
			title: "uppercase TRUE",
			in:    "TRUE",
			out:   true,
		},
		{
			title: "uppercase FALSE",
			in:    "FALSE",
			out:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				WebhookEnableAttributesDtKubernetes: c.in,
			}}

			out := ff.EnableAttributesDtKubernetes()

			assert.Equal(t, c.out, out)
		})
	}
}
