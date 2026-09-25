// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// testAppMountPath is where app mounts actually live since the overlay is mounted directly at the
// kubelet target path. It is deliberately outside of the CSI root dir.
const testAppMountPath = "/var/lib/kubelet/pods/6baaf3c7-1403-450b-b49a-86c08121a955/volumes/kubernetes.io~csi/oneagent-bin/mount"

const testCodeModulesBase = "/data/codemodules"

// testCodeModuleDir is base64 encoded, so it ends in the padding character, which must survive parsing.
const testCodeModuleDir = testCodeModulesBase + "/cHVibGljLmVjci5hd3MvZHluYXRyYWNlL2R5bmF0cmFjZS1jb2RlbW9kdWxlczoxLjMzOS44NC4yMDI2MDkwMy0xNjMzMzc="

func newAppMountPoint(path, lowerDirOpt string) mount.MountPoint {
	return mount.MountPoint{
		Device: "overlay",
		Path:   path,
		Type:   "overlay",
		Opts: []string{
			lowerDirOpt,
			"upperdir=/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/var",
			"workdir=/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/work",
		},
	}
}

// containerRootMountPoint is the overlay of the container the code itself runs in. It is always
// present in the mount table and must never be considered relevant.
func containerRootMountPoint() mount.MountPoint {
	return mount.MountPoint{
		Device: "overlay",
		Path:   "/",
		Type:   "overlay",
		Opts: []string{
			"lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/72/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/71/fs",
			"upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/73/fs",
			"workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/73/work",
		},
	}
}

func TestGetRelevantOverlayMounts(t *testing.T) {
	t.Run("get only relevant mounts", func(t *testing.T) {
		baseFolder := "/test/folder"
		expectedPath := baseFolder + "/some/sub/folder"
		expectedUpperDir := "/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/var"
		expectedWorkDir := "/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/work"

		relevantMountPoint := mount.MountPoint{
			Device: "overlay",
			Path:   expectedPath,
			Type:   "overlay",
			Opts: []string{
				"lowerdir=" + testCodeModuleDir,
				"upperdir=" + expectedUpperDir,
				"workdir=" + expectedWorkDir,
			},
		}

		mounter := mount.NewFakeMounter([]mount.MountPoint{
			relevantMountPoint,
			{
				Device: "not-relevant-mount-type",
			},
			{
				Device: "overlay",
				Path:   "not-relevant-overlay-mount",
				Type:   "overlay",
			},
		})

		mounts, err := GetOverlayMountsIn(t.Context(), mounter, baseFolder)
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Len(t, mounts, 1)
		assert.Equal(t, expectedPath, mounts[0].Path)
		assert.Equal(t, []string{testCodeModuleDir}, mounts[0].LowerDirs)
		assert.Equal(t, expectedUpperDir, mounts[0].UpperDir)
		assert.Equal(t, expectedWorkDir, mounts[0].WorkDir)
	})

	t.Run("works with no mount points", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		mounts, err := GetOverlayMountsIn(t.Context(), mounter, "/test")
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Empty(t, mounts)
	})

	t.Run("ignores irrelevant mounts", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{
				Device: "not-relevant-mount-type",
			},
			{
				Device: "overlay",
				Path:   "not-relevant-overlay-mount",
				Type:   "overlay",
			},
		})
		mounts, err := GetOverlayMountsIn(t.Context(), mounter, "/test")
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Empty(t, mounts)
	})

	t.Run("app mounts at the kubelet target path are not mounted under the CSI root", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, "lowerdir="+testCodeModuleDir),
		})

		mounts, err := GetOverlayMountsIn(t.Context(), mounter, "/data")
		require.NoError(t, err)
		assert.Empty(t, mounts)
	})

	t.Run("matches whole path components only", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint("/database/something", "lowerdir="+testCodeModuleDir),
		})

		mounts, err := GetOverlayMountsIn(t.Context(), mounter, "/data")
		require.NoError(t, err)
		assert.Empty(t, mounts)
	})
}

func TestGetOverlayMountsWithLowerDirIn(t *testing.T) {
	t.Run("finds app mount that is mounted outside of the CSI root", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, "lowerdir="+testCodeModuleDir),
			containerRootMountPoint(),
			{Device: "not-relevant-mount-type"},
		})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		require.Len(t, mounts, 1)
		assert.Equal(t, testAppMountPath, mounts[0].Path)
		assert.Equal(t, []string{testCodeModuleDir}, mounts[0].LowerDirs)
	})

	t.Run("ignores the overlay of the container itself", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{containerRootMountPoint()})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		assert.Empty(t, mounts)
	})

	t.Run("splits colon separated lower dirs", func(t *testing.T) {
		otherDir := testCodeModulesBase + "/1.2.3"
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, "lowerdir="+testCodeModuleDir+":"+otherDir),
		})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		require.Len(t, mounts, 1)
		assert.Equal(t, []string{testCodeModuleDir, otherDir}, mounts[0].LowerDirs)
	})

	t.Run("is relevant if any lower dir is under the base folder", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, "lowerdir=/somewhere/else:"+testCodeModuleDir),
		})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		require.Len(t, mounts, 1)
		assert.Equal(t, []string{"/somewhere/else", testCodeModuleDir}, mounts[0].LowerDirs)
	})

	t.Run("understands the lowerdir+ option of newer kernels", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, "lowerdir+="+testCodeModuleDir),
		})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		require.Len(t, mounts, 1)
		assert.Equal(t, []string{testCodeModuleDir}, mounts[0].LowerDirs)
	})

	t.Run("keeps escaped separators in a lower dir", func(t *testing.T) {
		escapedDir := testCodeModulesBase + "/weird:name"
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, `lowerdir=`+testCodeModulesBase+`/weird\:name`),
		})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		require.Len(t, mounts, 1)
		assert.Equal(t, []string{escapedDir}, mounts[0].LowerDirs)
	})

	t.Run("matches whole path components only", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			newAppMountPoint(testAppMountPath, "lowerdir=/data/codemodules-backup/1.2.3"),
		})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		assert.Empty(t, mounts)
	})

	t.Run("works with no mount points", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})

		mounts, err := GetOverlayMountsWithLowerDirIn(t.Context(), mounter, testCodeModulesBase)
		require.NoError(t, err)
		require.NotNil(t, mounts)
		assert.Empty(t, mounts)
	})
}

func TestGetRelevantDynaKubes(t *testing.T) {
	makeDK := func(name string, oaSpec oneagent.Spec) dynakube.DynaKube {
		return dynakube.DynaKube{
			Name: name, Namespace: "dynatrace",
			Spec: dynakube.DynaKubeSpec{OneAgent: oaSpec},
		}
	}

	reader := buildReader(t,
		makeDK("cloudnative-fullstack", oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{}}),
		makeDK("classic-fullstack", oneagent.Spec{ClassicFullStack: &oneagent.HostInjectSpec{}}),
		makeDK("host-monitoring", oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}}),
		makeDK("non-relevant", oneagent.Spec{}),
		makeDK("application-monitoring", oneagent.Spec{ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}}),
	)

	dks, err := GetRelevantDynaKubes(t.Context(), reader)
	require.NoError(t, err)

	expect := []string{
		"application-monitoring",
		"cloudnative-fullstack",
		"host-monitoring",
	}
	assert.Len(t, dks, len(expect))
	for _, dk := range dks {
		// Fake client will modify objects, so compare the name instead.
		assert.Contains(t, expect, dk.Name, "unexpected DynaKube name")
	}
}

func TestMigrateAppMounts(t *testing.T) {
	tempDir := t.TempDir()
	volID := "someid"

	checker := NewCorrectnessChecker(nil, dtcsi.CSIOptions{RootDir: tempDir})
	checker.mounter = mount.NewFakeMounter([]mount.MountPoint{
		{
			Device: "overlay",
			Path:   filepath.Join(tempDir, "appvol", volID, "mount"),
			Type:   "overlay",
		},
	})

	checker.migrateAppMounts(setupLogForTest(t))

	assert.DirExists(t, checker.path.AppMountsBaseDir())
	assert.FileExists(t, checker.path.AppMountForID(volID))
}

func TestMigrateHostMounts(t *testing.T) {
	apiReader := buildReader(t,
		dynakube.DynaKube{
			Name: "skip", Namespace: "dynatrace",
			Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}}},
		},
		dynakube.DynaKube{
			Name: "test", Namespace: "dynatrace",
			Spec: dynakube.DynaKubeSpec{
				APIURL:   "/e/tenant/api", // UUID: tenant
				OneAgent: oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}},
			},
		},
	)

	t.Run("skip agent dir exists", func(t *testing.T) {
		tempDir := t.TempDir()
		checker := NewCorrectnessChecker(apiReader, dtcsi.CSIOptions{RootDir: tempDir})
		require.NoError(t, os.MkdirAll(checker.path.OSAgentDir("test"), os.ModePerm))

		checker.migrateHostMounts(setupLogForTest(t))
	})

	t.Run("create symlink", func(t *testing.T) {
		tempDir := t.TempDir()
		checker := NewCorrectnessChecker(apiReader, dtcsi.CSIOptions{RootDir: tempDir})
		require.NoError(t, os.MkdirAll(checker.path.OldOSAgentDir("tenant"), os.ModePerm))

		checker.migrateHostMounts(setupLogForTest(t))

		assert.FileExists(t, checker.path.OSAgentDir("test")) // file because symlink
	})

	t.Run("skip missing source dir", func(t *testing.T) {
		tempDir := t.TempDir()
		checker := NewCorrectnessChecker(apiReader, dtcsi.CSIOptions{RootDir: tempDir})

		checker.migrateHostMounts(setupLogForTest(t))

		assert.NoFileExists(t, checker.path.OSAgentDir("test"))
		assert.NoDirExists(t, checker.path.OSAgentDir("test"))
	})
}

func buildReader(t *testing.T, dks ...dynakube.DynaKube) client.Reader {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, latest.AddToScheme(scheme))
	objs := make([]client.Object, len(dks))
	for i, dk := range dks {
		objs[i] = &dk
	}

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		Build()
}

func setupLogForTest(t *testing.T) context.Context {
	t.Helper()
	base := logd.Get()
	instrumented := logd.Logger{Logger: logd.Get().WithSink(testFailingLogSink{LogSink: base.GetSink(), t: t})}

	return logd.IntoContext(t.Context(), instrumented)
}

type testFailingLogSink struct {
	logr.LogSink
	t *testing.T
}

var _ logr.LogSink = testFailingLogSink{}

func (t testFailingLogSink) WithName(name string) logr.LogSink            { return t }
func (t testFailingLogSink) WithValues(keysAndValues ...any) logr.LogSink { return t }
func (t testFailingLogSink) Error(err error, msg string, keysAndValues ...any) {
	t.LogSink.Error(err, msg, keysAndValues...)
	t.t.FailNow()
}
