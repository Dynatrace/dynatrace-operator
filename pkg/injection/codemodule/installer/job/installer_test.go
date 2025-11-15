package job

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
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

	t.Run("returns early if path doesn't exist", func(t *testing.T) {
		path := metadata.PathResolver{RootDir: t.TempDir()}
		installer := Installer{
			props: &Properties{
				PathResolver: path,
				ImageURI:     testImageURL,
			},
		}
		isDownloaded := installer.isAlreadyPresent(path.AgentSharedBinaryDirForAgent(testVersion))
		assert.False(t, isDownloaded)
	})
	t.Run("returns true if path present", func(t *testing.T) {
		path := metadata.PathResolver{RootDir: t.TempDir()}
		_ = os.MkdirAll(path.AgentSharedBinaryDirForAgent(testVersion), 0777)

		installer := Installer{
			props: &Properties{
				PathResolver: path,
			},
		}
		isDownloaded := installer.isAlreadyPresent(path.AgentSharedBinaryDirForAgent(testVersion))
		assert.True(t, isDownloaded)
	})
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
	nodeName := "node-1"

	t.Run("nothing present -> create job", func(t *testing.T) {
		cl := fake.NewClient()
		props := &Properties{
			APIReader:    cl,
			Client:       cl,
			Owner:        &owner,
			PathResolver: metadata.PathResolver{RootDir: t.TempDir()},
		}
		inst := &Installer{
			nodeName: nodeName,
			props:    props,
		}

		targetDir := filepath.Join(t.TempDir(), "download", "here", "1.2.3")

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
			APIReader:    setupInCompleteJob(t, name, owner.Namespace),
			Client:       cl,
			Owner:        &owner,
			PathResolver: metadata.PathResolver{RootDir: t.TempDir()},
		}
		inst := &Installer{
			nodeName: nodeName,
			props:    props,
		}

		targetDir := filepath.Join(t.TempDir(), "download", "here", "1.2.3")

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
		path := metadata.PathResolver{RootDir: t.TempDir()}
		props := &Properties{
			APIReader:    cl,
			Client:       cl,
			Owner:        &owner,
			PathResolver: path,
		}
		inst := &Installer{
			nodeName: nodeName,
			props:    props,
		}

		targetDir := filepath.Join(t.TempDir(), "download", "here", "1.2.3")

		require.NoError(t, os.MkdirAll(targetDir, os.ModePerm))

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
