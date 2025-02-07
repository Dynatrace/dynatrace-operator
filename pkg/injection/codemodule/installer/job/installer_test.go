package job

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIsAlreadyPresent(t *testing.T) {
	testImageURL := "test:5000/repo@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	testVersion := "1.2.3"
	pathResolver := metadata.PathResolver{}

	t.Run("returns early if path doesn't exist", func(t *testing.T) {
		installer := Installer{
			fs: afero.NewMemMapFs(),
			props: &Properties{
				PathResolver: pathResolver,
				ImageUri:     testImageURL,
			},
		}
		isDownloaded := installer.isAlreadyPresent(pathResolver.AgentSharedBinaryDirForAgent(testVersion))
		assert.False(t, isDownloaded)
	})
	t.Run("returns true if path present", func(t *testing.T) {
		installer := Installer{
			fs: testFileSystemWithSharedDirPresent(pathResolver, testVersion),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyPresent(pathResolver.AgentSharedBinaryDirForAgent(testVersion))
		assert.True(t, isDownloaded)
	})
}

func testFileSystemWithSharedDirPresent(pathResolver metadata.PathResolver, imageDigest string) afero.Fs {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll(pathResolver.AgentSharedBinaryDirForAgent(imageDigest), 0777)

	return fs
}

func TestIsReady(t *testing.T) {
	ctx := context.Background()
	owner := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test",
		},
	}
	name := "job-1"
	targetDir := "/download/here/1.2.3"
	nodeName := "node-1"

	t.Run("nothing present -> create job", func(t *testing.T) {
		cl := fake.NewClient()
		props := &Properties{
			ApiReader: cl,
			Client:    cl,
			Owner:     &owner,
		}
		inst := &Installer{
			fs:       afero.NewMemMapFs(),
			nodeName: nodeName,
			props:    props,
		}

		ready, err := inst.isReady(ctx, targetDir, name)
		require.NoError(t, err)
		require.False(t, ready)

		var newJob batchv1.Job

		require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: name, Namespace: owner.GetNamespace()}, &newJob))
		require.NotEmpty(t, newJob)
	})

	t.Run("job present, target not present -> return not ready, no error", func(t *testing.T) {
		cl := fake.NewClient()
		props := &Properties{
			ApiReader: setupInCompleteJob(t, name, owner.Namespace),
			Client:    cl,
			Owner:     &owner,
		}
		inst := &Installer{
			fs:       afero.NewMemMapFs(),
			nodeName: nodeName,
			props:    props,
		}

		ready, err := inst.isReady(ctx, targetDir, name)
		require.NoError(t, err)
		require.False(t, ready)

		var newJob batchv1.Job // should not create a Job as it already exists, bit of a hack, that I use different clients for ApiReader and Client
		err = cl.Get(ctx, types.NamespacedName{Name: name, Namespace: owner.GetNamespace()}, &newJob)
		require.Error(t, err)
		require.NoError(t, client.IgnoreNotFound(err))
	})

	t.Run("job present, target present -> return ready, cleanup job", func(t *testing.T) {
		cl := setupCompleteJob(t, name, owner.Namespace)
		props := &Properties{
			ApiReader: cl,
			Client:    cl,
			Owner:     &owner,
		}
		inst := &Installer{
			fs:       afero.NewMemMapFs(),
			nodeName: nodeName,
			props:    props,
		}

		setupTargetDir(t, inst.fs, targetDir)

		ready, err := inst.isReady(ctx, targetDir, name)
		require.NoError(t, err)
		require.True(t, ready)

		var newJob batchv1.Job
		err = cl.Get(ctx, types.NamespacedName{Name: name, Namespace: owner.GetNamespace()}, &newJob)
		require.Error(t, err)
		require.NoError(t, client.IgnoreNotFound(err))
	})
}

func setupCompleteJob(t *testing.T, name, namespace string) client.Client {
	t.Helper()

	fakeJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	return fake.NewClient(&fakeJob)
}

func setupInCompleteJob(t *testing.T, name, namespace string) client.Client {
	t.Helper()

	fakeJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: batchv1.JobStatus{
			Succeeded: 0,
		},
	}

	return fake.NewClient(&fakeJob)
}

func setupTargetDir(t *testing.T, fs afero.Fs, targetDir string) {
	t.Helper()

	require.NoError(t, fs.MkdirAll(targetDir, os.ModePerm))
}
