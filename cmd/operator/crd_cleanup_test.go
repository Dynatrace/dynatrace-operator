package operator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/projectpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join(projectpath.Root, "bin", "k8s")

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}

	return ""
}

func addFirstFoundEnvTestBinaryDir(t *testing.T, env *envtest.Environment) {
	binDir := getFirstFoundEnvTestBinaryDir()
	if binDir != "" {
		env.BinaryAssetsDirectory = binDir
		t.Logf("using envtest binary assets from: %s", binDir)
	}
}

func TestCleanupCRDStorageVersions(t *testing.T) {
	t.Run("no cleanup when CRD doesn't exist", func(t *testing.T) {
		testEnv := &envtest.Environment{}
		addFirstFoundEnvTestBinaryDir(t, testEnv)

		cfg, err := testEnv.Start()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, testEnv.Stop())
		}()

		clt, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
		require.NoError(t, err)

		err = performCRDStorageVersionsCleanup(context.Background(), clt)
		assert.NoError(t, err)
	})

	t.Run("no cleanup when single storage version", func(t *testing.T) {
		testEnv := &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join(projectpath.Root, "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}
		addFirstFoundEnvTestBinaryDir(t, testEnv)

		cfg, err := testEnv.Start()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, testEnv.Stop())
		}()

		clt, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
		require.NoError(t, err)

		ctx := context.Background()

		var crd apiextensionsv1.CustomResourceDefinition
		err = clt.Get(ctx, client.ObjectKey{Name: k8scrd.DynaKubeName}, &crd)
		require.NoError(t, err)

		crd.Status.StoredVersions = []string{"v1beta6"}
		err = clt.Status().Update(ctx, &crd)
		require.NoError(t, err)

		err = performCRDStorageVersionsCleanup(ctx, clt)
		require.NoError(t, err)

		err = clt.Get(ctx, client.ObjectKey{Name: k8scrd.DynaKubeName}, &crd)
		require.NoError(t, err)
		assert.Equal(t, []string{"v1beta6"}, crd.Status.StoredVersions)
	})

	t.Run("cleanup when multiple storage versions", func(t *testing.T) {
		testEnv := &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join(projectpath.Root, "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}
		addFirstFoundEnvTestBinaryDir(t, testEnv)

		cfg, err := testEnv.Start()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, testEnv.Stop())
		}()

		clt, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
		require.NoError(t, err)

		ctx := context.Background()

		var crd apiextensionsv1.CustomResourceDefinition
		err = clt.Get(ctx, client.ObjectKey{Name: k8scrd.DynaKubeName}, &crd)
		require.NoError(t, err)

		crd.Status.StoredVersions = []string{"v1beta5", "v1beta6"}
		err = clt.Status().Update(ctx, &crd)
		require.NoError(t, err)

		// Create version older than latest
		dk1 := &v1beta5.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk-1",
				Namespace: k8senv.DefaultNamespace(),
			},
			Spec: v1beta5.DynaKubeSpec{
				APIURL: "https://test.dynatrace.com/api",
			},
		}

		dk2 := &v1beta5.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk-2",
				Namespace: k8senv.DefaultNamespace(),
			},
			Spec: v1beta5.DynaKubeSpec{
				APIURL: "https://test2.dynatrace.com/api",
			},
		}

		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: k8senv.DefaultNamespace(),
			},
		}
		err = clt.Create(ctx, &ns)
		require.NoError(t, err)

		err = clt.Create(ctx, dk1)
		require.NoError(t, err)
		err = clt.Create(ctx, dk2)
		require.NoError(t, err)

		err = performCRDStorageVersionsCleanup(ctx, clt)
		require.NoError(t, err)

		err = clt.Get(ctx, client.ObjectKey{Name: k8scrd.DynaKubeName}, &crd)
		require.NoError(t, err)
		assert.Equal(t, []string{"v1beta6"}, crd.Status.StoredVersions)

		var dynakubeList dynakube.DynaKubeList
		err = clt.List(ctx, &dynakubeList)
		require.NoError(t, err)
		assert.Len(t, dynakubeList.Items, 2)
	})
}
