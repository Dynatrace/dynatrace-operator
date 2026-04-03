package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sversion "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestGetServerVersion(t *testing.T) {
	fakeClientSet := fakeclientset.NewClientset()
	fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &k8sversion.Info{
		Major:      "1",
		Minor:      "34",
		GitVersion: "v1.34.4-gke.1047000",
		Platform:   "linux/amd64",
	}

	serverVersion, err := GetServerVersion(fakeDiscovery)
	require.NoError(t, err)
	assert.Equal(t, "1", serverVersion.Major)
	assert.Equal(t, "34", serverVersion.Minor)
	assert.Equal(t, "v1.34.4-gke.1047000", serverVersion.GitVersion)
	assert.Equal(t, "linux/amd64", serverVersion.Platform)
}

func TestGetFormattedServerVersion(t *testing.T) {
	fakeClientSet := fakeclientset.NewClientset()
	fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &k8sversion.Info{
		Major:      "1",
		Minor:      "34",
		GitVersion: "v1.34.4-gke.1047000",
		GitCommit:  "f726e501477d4f55aed8a5102d5cd49de506272b",
		BuildDate:  "2026-02-10T20:20:49Z",
		GoVersion:  "go1.24.12",
		Platform:   "linux/amd64",
	}

	result, err := GetFormattedServerVersion(fakeDiscovery)
	require.NoError(t, err)

	assert.Contains(t, result, "major: 1")
	assert.Contains(t, result, "minor: 34")
	assert.Contains(t, result, "gitVersion: v1.34.4-gke.1047000")
	assert.Contains(t, result, "gitCommit: f726e501477d4f55aed8a5102d5cd49de506272b")
	assert.Contains(t, result, "buildDate: 2026-02-10T20:20:49Z")
	assert.Contains(t, result, "goVersion: go1.24.12")
	assert.Contains(t, result, "platform: linux/amd64")
}
