package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAgentInitialConnectRetry(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   int
		istio bool
	}

	cases := []testCase{
		{
			title: "default, no istio",
			in:    "",
			out:   -1,
			istio: false,
		},
		{
			title: "default, with istio",
			in:    "",
			out:   DefaultOAIstioInitialConnectRetry,
			istio: true,
		},
		{
			title: "overrule, no istio",
			in:    "11",
			out:   11,
			istio: false,
		},
		{
			title: "overrule, with istio",
			in:    "12",
			out:   12,
			istio: true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				OAInitialConnectRetryKey: c.in,
			}}

			out := ff.GetAgentInitialConnectRetry(c.istio)

			assert.Equal(t, c.out, out)
		})
	}
}

func TestGetNodeImagePullTechnology(t *testing.T) {
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
			in:    "tech1,tech2",
			out:   "tech1,tech2",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				OANodeImagePullTechnologiesKey: c.in,
			}}

			out := ff.GetNodeImagePullTechnology()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestGetOneAgentMaxUnavailable(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   int
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   1,
		},
		{
			title: "overrule",
			in:    "3",
			out:   3,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				OAMaxUnavailableKey: c.in,
			}}

			out := ff.GetOneAgentMaxUnavailable()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestOneAgentIgnoresProxyProxy(t *testing.T) {
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
				OAProxyIgnoredKey: c.in,
			}}

			out := ff.OneAgentIgnoresProxy()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsOneAgentPrivileged(t *testing.T) {
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
				OAPrivilegedKey: c.in,
			}}

			out := ff.IsOneAgentPrivileged()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestSkipOneAgentLivenessProbe(t *testing.T) {
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
				OASkipLivenessProbeKey: c.in,
			}}

			out := ff.SkipOneAgentLivenessProbe()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsNodeImagePull(t *testing.T) {
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
				OANodeImagePullKey: c.in,
			}}

			out := ff.IsNodeImagePull()

			assert.Equal(t, c.out, out)
		})
	}
}
