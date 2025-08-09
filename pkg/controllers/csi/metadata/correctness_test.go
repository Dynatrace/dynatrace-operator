package metadata

import (
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetRelevantOverlayMounts(t *testing.T) {
	t.Run("get only relevant mounts", func(t *testing.T) {
		baseFolder := "/test/folder"
		expectedPath := baseFolder + "/some/sub/folder"
		expectedLowerDir := "/data/codemodules/cXVheS5pby9keW5hdHJhY2UvZHluYXRyYWNlLWJvb3RzdHJhcHBlcjpzbmFwc2hvdA=="
		expectedUpperDir := "/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/var"
		expectedWorkDir := "/data/appmounts/csi-a3dd8a9ab6e64e92efca99a0d180da60ab807f0e31a04e11edb451311130211c/work"

		relevantMountPoint := mount.MountPoint{
			Device: "overlay",
			Path:   expectedPath,
			Type:   "overlay",
			Opts: []string{
				"lowerdir=" + expectedLowerDir,
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

		mounts, err := GetRelevantOverlayMounts(mounter, baseFolder)
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Len(t, mounts, 1)
		assert.Equal(t, expectedPath, mounts[0].Path)
		assert.Equal(t, expectedLowerDir, mounts[0].LowerDir)
		assert.Equal(t, expectedUpperDir, mounts[0].UpperDir)
		assert.Equal(t, expectedWorkDir, mounts[0].WorkDir)
	})

	t.Run("works with no mount points", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		mounts, err := GetRelevantOverlayMounts(mounter, "")
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
		mounts, err := GetRelevantOverlayMounts(mounter, "/test")
		require.NoError(t, err)
		require.NotNil(t, mounts)
		require.Empty(t, mounts)
	})
}

func TestGetRelevantDynaKubes(t *testing.T) {
	makeDK := func(name string, oaSpec oneagent.Spec) dynakube.DynaKube {
		return dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
			Spec:       dynakube.DynaKubeSpec{OneAgent: oaSpec},
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

	setupLogForTest(t)
	checker.migrateAppMounts()

	assert.DirExists(t, checker.path.AppMountsBaseDir())
	assert.FileExists(t, checker.path.AppMountForID(volID))
}

func TestMigrateHostMounts(t *testing.T) {
	apiReader := buildReader(t,
		dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "skip", Namespace: "dynatrace"},
			Spec:       dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}}},
		},
		dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "dynatrace"},
			Spec: dynakube.DynaKubeSpec{
				APIURL:   "/e/tenant/api", // UUID: tenant
				OneAgent: oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}},
			},
		},
	)

	t.Run("skip agent dir exists", func(t *testing.T) {
		tempDir := t.TempDir()
		checker := NewCorrectnessChecker(apiReader, dtcsi.CSIOptions{RootDir: tempDir})
		require.NoError(t, os.MkdirAll(checker.path.OsAgentDir("test"), os.ModePerm))

		setupLogForTest(t)
		checker.migrateHostMounts(t.Context())
	})

	t.Run("create symlink", func(t *testing.T) {
		tempDir := t.TempDir()
		checker := NewCorrectnessChecker(apiReader, dtcsi.CSIOptions{RootDir: tempDir})
		require.NoError(t, os.MkdirAll(checker.path.OldOsAgentDir("tenant"), os.ModePerm))

		setupLogForTest(t)
		checker.migrateHostMounts(t.Context())

		assert.FileExists(t, checker.path.OsAgentDir("test")) // file because symlink
	})

	t.Run("skip missing source dir", func(t *testing.T) {
		tempDir := t.TempDir()
		checker := NewCorrectnessChecker(apiReader, dtcsi.CSIOptions{RootDir: tempDir})

		setupLogForTest(t)
		checker.migrateHostMounts(t.Context())

		assert.NoFileExists(t, checker.path.OsAgentDir("test"))
		assert.NoDirExists(t, checker.path.OsAgentDir("test"))
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

func setupLogForTest(t *testing.T) {
	oldLog := log
	log = logd.Logger{Logger: log.WithSink(testFailingLogSink{LogSink: log.GetSink(), t: t})}
	t.Cleanup(func() { log = oldLog })
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
