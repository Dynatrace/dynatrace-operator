package supportarchive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesVersionCollector(t *testing.T) {
	logBuffer := bytes.Buffer{}
	buffer := bytes.Buffer{}
	archive := newZipArchive(bufio.NewWriter(&buffer))

	fakeClientSet := fakeclientset.NewClientset()
	fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		Major:        "1",
		Minor:        "34",
		GitVersion:   "v1.34.4-gke.1047000",
		GitCommit:    "f726e501477d4f55aed8a5102d5cd49de506272b",
		GitTreeState: "clean",
		BuildDate:    "2026-01-01T13:37:00Z",
		Platform:     "linux/amd64/go1.24.12 X:boringcrypto",
	}

	versionCollector := newKubernetesVersionCollector(newSupportArchiveLogger(&logBuffer), archive, fakeDiscovery)
	assert.Equal(t, kubernetesVersionCollectorName, versionCollector.Name())

	require.NoError(t, versionCollector.Do())
	require.NoError(t, archive.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	assert.Len(t, zipReader.File, 1)

	file := zipReader.File[0]
	assert.Equal(t, KubernetesVersionFileName, file.Name)

	reader, err := file.Open()
	require.NoError(t, err)
	defer assertNoErrorOnClose(t, reader)

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	versionContent := string(content)

	assert.Contains(t, logBuffer.String(), "Storing Kubernetes version")
	assert.Contains(t, versionContent, "minor: 34")
	assert.Contains(t, versionContent, "gitVersion: v1.34.4-gke.1047000")
	assert.Contains(t, versionContent, "platform: linux/amd64/go1.24.12 X:boringcrypto")
}
