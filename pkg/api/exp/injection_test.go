package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIgnoredNamespaces(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   []string
	}

	ns := "own-ns"
	defaultIgnored := getDefaultIgnoredNamespaces(ns)

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   defaultIgnored,
		},
		{
			title: "incorrect input",
			in:    "[1,2,3]",
			out:   defaultIgnored,
		},
		{
			title: "overrule",
			in:    "[\"^dynatrace$\", \"openshift(-.*)?\"]",
			out:   []string{"^dynatrace$", "openshift(-.*)?"},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				InjectionIgnoredNamespacesKey: c.in,
			}}

			out := ff.GetIgnoredNamespaces(ns)

			assert.Equal(t, c.out, out)
		})
	}
}

func TestGetInjectionFailurePolicy(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   string
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   silentPhrase,
		},
		{
			title: "incorrect input",
			in:    "flail",
			out:   silentPhrase,
		},
		{
			title: "overrule",
			in:    failPhrase,
			out:   failPhrase,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				InjectionFailurePolicyKey: c.in,
			}}

			out := ff.GetInjectionFailurePolicy()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsInjectionAutomatic(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   bool
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   true,
		},
		{
			title: "overrule",
			in:    "false",
			out:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				InjectionAutomaticKey: c.in,
			}}

			out := ff.IsAutomaticInjection()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsLabelVersionDetection(t *testing.T) {
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
				InjectionLabelVersionDetectionKey: c.in,
			}}

			out := ff.IsLabelVersionDetection()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestHasInitSeccomp(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   bool
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			out:   true,
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
				InjectionSeccompKey: c.in,
			}}

			out := ff.HasInitSeccomp()

			assert.Equal(t, c.out, out)
		})
	}
}
