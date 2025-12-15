//go:build e2e

package operator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

func TestGetHelmOptions(t *testing.T) {
	assertOptions := func(t *testing.T, expect *helm.Opts, options []helm.Option) {
		t.Helper()
		got := new(helm.Opts)
		for _, opt := range options {
			opt(got)
		}
		assert.Equal(t, expect, got)
	}

	t.Run("use release tag", func(t *testing.T) {
		opts, err := getHelmOptions("1.2.3", "test", true)
		require.NoError(t, err)
		assertOptions(t, &helm.Opts{
			Namespace:   "dynatrace",
			ReleaseName: "dynatrace-operator",
			Version:     "1.2.3",
			Args: []string{
				"--create-namespace",
				"--install",
				"--set", "platform=test",
				"--set", "installCRD=true",
				"--set", "csidriver.enabled=true",
				"--set", "manifests=true",
				"--set", "debugLogs=true",
				helmRegistryURL,
			},
		}, opts)
	})

	t.Run("use nightly", func(t *testing.T) {
		t.Setenv("HELM_CHART", "oci://registry:0.0.0-nightly-chart")
		opts, err := getHelmOptions("", "test", true)
		require.NoError(t, err)
		assertOptions(t, &helm.Opts{
			Namespace:   "dynatrace",
			ReleaseName: "dynatrace-operator",
			Args: []string{
				"--create-namespace",
				"--install",
				"--set", "platform=test",
				"--set", "installCRD=true",
				"--set", "csidriver.enabled=true",
				"--set", "manifests=true",
				"--set", "debugLogs=true",
				"oci://registry:0.0.0-nightly-chart",
			},
		}, opts)
	})

	t.Run("use filesystem", func(t *testing.T) {
		tempDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "make"), []byte("#!/bin/sh\necho repo:tag"), os.ModePerm)) //nolint:gosec
		t.Setenv("PATH", tempDir+":"+os.Getenv("PATH"))

		t.Setenv("HELM_CHART", "oci://registry:snapshot-test")
		opts, err := getHelmOptions("", "test", false)
		require.NoError(t, err)
		assertOptions(t, &helm.Opts{
			Namespace:   "dynatrace",
			ReleaseName: "dynatrace-operator",
			Args: []string{
				"--create-namespace",
				"--install",
				"--set", "platform=test",
				"--set", "installCRD=true",
				"--set", "csidriver.enabled=false",
				"--set", "manifests=true",
				"--set", "debugLogs=true",
				"--set", "image=repo:tag",
				filepath.Join(project.RootDir(), "config", "helm", "chart", "default"),
			},
		}, opts)

		// "make" should fail if reinvoked
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "make"), []byte("#!/bin/sh\nexit 1"), os.ModePerm)) //nolint:gosec

		opts, err = getHelmOptions("", "test", false)
		require.NoError(t, err)
		assertOptions(t, &helm.Opts{
			Namespace:   "dynatrace",
			ReleaseName: "dynatrace-operator",
			Args: []string{
				"--create-namespace",
				"--install",
				"--set", "platform=test",
				"--set", "installCRD=true",
				"--set", "csidriver.enabled=false",
				"--set", "manifests=true",
				"--set", "debugLogs=true",
				"--set", "image=repo:tag",
				filepath.Join(project.RootDir(), "config", "helm", "chart", "default"),
			},
		}, opts)
	})

	t.Run("no image found", func(t *testing.T) {
		tempDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "make"), []byte("#!/bin/sh\necho make[1] Entering directory"), os.ModePerm)) //nolint:gosec
		t.Setenv("PATH", tempDir+":"+os.Getenv("PATH"))

		_, err := getHelmOptions("", "test", false)
		require.Error(t, err)
	})
}
