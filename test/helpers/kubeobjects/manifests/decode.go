//go:build e2e

package manifests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/decoder"
)

func ObjectFromFile[T client.Object](t *testing.T, path string) T {
	kubernetesManifest, err := os.Open(path)
	defer func() { require.NoError(t, kubernetesManifest.Close()) }()
	require.NoError(t, err)

	rawObject, err := decoder.DecodeAny(kubernetesManifest)
	require.NoError(t, err)

	object, ok := rawObject.(T)
	require.True(t, ok)

	return object
}
