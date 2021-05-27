package csidriver

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/mount"
)

type errorMounter struct {
	*mount.FakeMounter
	errorMap map[string]error
}

func newErrorMounter(errorMap map[string]error) *errorMounter {
	if errorMap == nil {
		errorMap = map[string]error{}
	}
	return &errorMounter{
		FakeMounter: mount.NewFakeMounter([]mount.MountPoint{}),
		errorMap:    errorMap,
	}
}

func (mounter errorMounter) Mount(source string, target string, fsType string, options []string) error {
	err, ok := mounter.errorMap[source]
	if ok {
		return err
	}
	return mounter.FakeMounter.Mount(source, target, fsType, options)
}

func TestBindMount(t *testing.T) {
	t.Run(`Bind nothing`, func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpRootDir := path.Join(tmpDir, "test")

		var mounts []Mount
		err := bindMount(&bindOptions{
			mounter: mount.NewFakeMounter([]mount.MountPoint{}),
			mounts:  mounts,
			rootDir: tmpRootDir,
		})
		assert.NoError(t, err)
	})
	t.Run(`Bind read-only mount`, func(t *testing.T) {
		tmpDir := t.TempDir()
		source := path.Join(tmpDir, "source")
		target := path.Join(tmpDir, "target")

		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		mounts := []Mount{
			{Source: source, Target: target, ReadOnly: true},
		}
		err := bindMount(&bindOptions{
			mounter: mounter,
			mounts:  mounts,
			rootDir: tmpDir,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(mounter.MountPoints))
		assert.Equal(t, source, mounter.MountPoints[0].Device)
		assert.Equal(t, target, mounter.MountPoints[0].Path)
		assert.Equal(t, 2, len(mounter.MountPoints[0].Opts))
		assert.Contains(t, mounter.MountPoints[0].Opts, "bind")
		assert.Contains(t, mounter.MountPoints[0].Opts, "ro")
	})
	t.Run(`Bind normal mount`, func(t *testing.T) {
		tmpDir := t.TempDir()
		source := path.Join(tmpDir, "source")
		target := path.Join(tmpDir, "target")

		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		mounts := []Mount{
			{Source: source, Target: target, ReadOnly: false},
		}
		err := bindMount(&bindOptions{
			mounter: mounter,
			mounts:  mounts,
			rootDir: tmpDir,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(mounter.MountPoints))
		assert.Equal(t, source, mounter.MountPoints[0].Device)
		assert.Equal(t, target, mounter.MountPoints[0].Path)
		assert.Equal(t, 1, len(mounter.MountPoints[0].Opts))
		assert.Contains(t, mounter.MountPoints[0].Opts, "bind")
		assert.NotContains(t, mounter.MountPoints[0].Opts, "ro")
	})
	t.Run(`Handle error on bind`, func(t *testing.T) {
		tmpDir := t.TempDir()
		source := path.Join(tmpDir, "source")
		target := path.Join(tmpDir, "target")

		mounter := newErrorMounter(
			map[string]error{
				source: fmt.Errorf("test-error"),
			})
		mounts := []Mount{
			{Source: source, Target: target, ReadOnly: false},
		}
		err := bindMount(&bindOptions{
			mounter: mounter,
			mounts:  mounts,
			rootDir: tmpDir,
		})

		assert.Error(t, err)
	})
	t.Run(`Handle error on bind after successful binds`, func(t *testing.T) {
		tmpDir := t.TempDir()
		source0 := path.Join(tmpDir, "source-0")
		target0 := path.Join(tmpDir, "target-0")
		source1 := path.Join(tmpDir, "source-1")
		target1 := path.Join(tmpDir, "target-1")

		mounter := newErrorMounter(
			map[string]error{
				source1: fmt.Errorf("test-error"),
			})
		mounts := []Mount{
			{Source: source0, Target: target0, ReadOnly: false},
			{Source: source1, Target: target1, ReadOnly: false},
		}
		err := bindMount(&bindOptions{
			mounter: mounter,
			mounts:  mounts,
			rootDir: tmpDir,
		})

		assert.Error(t, err)
		// First MountPoint still exists, because target dir does not exist and therefore unbind skips it
		assert.Equal(t, 1, len(mounter.MountPoints))
		assert.Equal(t, source0, mounter.MountPoints[0].Device)
		assert.Equal(t, target0, mounter.MountPoints[0].Path)
	})
}

func TestUnbindMount(t *testing.T) {
	t.Run(`handle unbind error`, func(t *testing.T) {
		tmpDir := t.TempDir()
		source0 := path.Join(tmpDir, "source-0")
		target0 := path.Join(tmpDir, "target-0")
		source1 := path.Join(tmpDir, "source-1")
		target1 := path.Join(tmpDir, "target-1")

		mountPoints := []mount.MountPoint{
			{Device: source0, Path: target0, Type: "", Opts: []string{"bind"}, Freq: 0, Pass: 0},
		}
		mounter := newErrorMounter(
			map[string]error{
				source1: fmt.Errorf("test-error"),
			})
		mounter.FakeMounter.MountCheckErrors = map[string]error{
			target0: fmt.Errorf("test-error"),
		}
		mounter.MountPoints = mountPoints
		mounts := []Mount{
			{Source: source0, Target: target0, ReadOnly: false},
			{Source: source1, Target: target1, ReadOnly: false},
		}
		err := bindUnmount(&bindOptions{
			mounter: mounter,
			mounts:  mounts,
			rootDir: tmpDir,
		})

		assert.Error(t, err)
		// First MountPoint still exists, because unmounting target dir produced error
		assert.Equal(t, 1, len(mounter.MountPoints))
		assert.Equal(t, source0, mounter.MountPoints[0].Device)
		assert.Equal(t, target0, mounter.MountPoints[0].Path)
	})
	t.Run(`unbind`, func(t *testing.T) {
		tmpDir, _ := filepath.EvalSymlinks(t.TempDir())
		source0 := path.Join(tmpDir, "source-0")
		target0 := path.Join(tmpDir, "target-0")
		source1 := path.Join(tmpDir, "source-1")
		target1 := path.Join(tmpDir, "target-1")

		// Make sure directories exist so no error is returned while unmounting
		err := os.MkdirAll(target0, os.ModeDir)
		require.NoError(t, err)

		err = os.MkdirAll(target1, os.ModeDir)
		require.NoError(t, err)

		mountPoints := []mount.MountPoint{
			{Device: source0, Path: target0, Type: "", Opts: []string{"bind"}, Freq: 0, Pass: 0},
		}
		mounter := newErrorMounter(
			map[string]error{
				source1: fmt.Errorf("test-error"),
			})
		mounter.MountPoints = mountPoints
		mounts := []Mount{
			{Source: source0, Target: target0, ReadOnly: false},
			{Source: source1, Target: target1, ReadOnly: false},
		}
		err = bindUnmount(&bindOptions{
			mounter: mounter,
			mounts:  mounts,
			rootDir: tmpDir,
		})

		assert.NoError(t, err)
		// Target dir exists, mounter could successfully unbind
		assert.Equal(t, 0, len(mounter.MountPoints))
	})
}
