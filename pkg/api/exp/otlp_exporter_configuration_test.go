package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureFlags_IsOTLPExporterConfiguration(t *testing.T) {
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
				OTLPExporterConfigurationKey: c.in,
			}}

			out := ff.IsOTLPExporterConfiguration()

			assert.Equal(t, c.out, out)
		})
	}
}
