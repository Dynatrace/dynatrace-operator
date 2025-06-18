package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsActiveGateUpdatesDisabled(t *testing.T) {
	type testCase struct {
		title string
		in    string
		depIn string
		out   bool
	}

	cases := []testCase{
		{
			title: "default",
			in:    "",
			depIn: "",
			out:   false,
		},
		{
			title: "overrule, non deprecated key",
			in:    "false",
			depIn: "",
			out:   true,
		},
		{
			title: "overrule, deprecated key",
			in:    "",
			depIn: "true",
			out:   true,
		},
		{
			title: "overrule, no deprecated key overrules",
			in:    "true",
			depIn: "true",
			out:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				AGUpdatesKey:        c.in,
				AGDisableUpdatesKey: c.depIn,
			}}

			out := ff.IsActiveGateUpdatesDisabled()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsActiveGateAutomaticTLSCertificate(t *testing.T) {
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
				AGAutomaticTLSCertificateKey: c.in,
			}}

			out := ff.IsActiveGateAutomaticTLSCertificate()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsAutomaticK8sApiMonitoring(t *testing.T) {
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
				AGAutomaticK8sAPIMonitoringKey: c.in,
			}}

			out := ff.IsAutomaticK8sAPIMonitoring()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestGetAutomaticK8sApiMonitoringClusterName(t *testing.T) {
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
			in:    "my-name",
			out:   "my-name",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ff := FeatureFlags{annotations: map[string]string{
				AGAutomaticK8sAPIMonitoringClusterNameKey: c.in,
			}}

			out := ff.GetAutomaticK8sAPIMonitoringClusterName()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsK8sAppEnabled(t *testing.T) {
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
				AGK8sAppEnabledKey: c.in,
			}}

			out := ff.IsK8sAppEnabled()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestIsActiveGateAppArmor(t *testing.T) {
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
				AGAppArmorKey: c.in,
			}}

			out := ff.IsActiveGateAppArmor()

			assert.Equal(t, c.out, out)
		})
	}
}

func TestAGIgnoresProxyProxy(t *testing.T) {
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
				AGIgnoreProxyKey: c.in,
			}}

			out := ff.AGIgnoresProxy()

			assert.Equal(t, c.out, out)
		})
	}
}
