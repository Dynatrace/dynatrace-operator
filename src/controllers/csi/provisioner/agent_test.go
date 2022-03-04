package csiprovisioner

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	t_utils "github.com/Dynatrace/dynatrace-operator/src/testing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

const (
	testVersion = "test"
)

func TestNewAgentUpdater(t *testing.T) {
	t.Run(`create`, func(t *testing.T) {
		createTestAgentUpdater(t, nil)
	})
}

func TestGetOneAgentVersionFromInstance(t *testing.T) {
	t.Run(`use status`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: testVersion,
			},
		}
		updater := createTestAgentUpdater(t, &dk)

		version := updater.getOneAgentVersionFromInstance()
		assert.Equal(t, testVersion, version)
	})
	t.Run(`use version `, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						Version: testVersion,
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: "other",
			},
		}
		updater := createTestAgentUpdater(t, &dk)

		version := updater.getOneAgentVersionFromInstance()
		assert.Equal(t, testVersion, version)
	})
}

func TestUpdateAgent(t *testing.T) {
	t.Run(`fresh install`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						Version: testVersion,
					},
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("1")
		previousHash := ""
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		updater.installer.(*installer.InstallerMock).
			On("SetVersion", testVersion).
			Return()
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(nil)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			previousHash,
			&processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)
		t_utils.AssertEvents(t,
			updater.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    installAgentVersionEvent,
				},
			},
		)
	})
	t.Run(`update`, func(t *testing.T) {
		updateOneagent(t, false)
	})
	t.Run(`update to already installed version`, func(t *testing.T) {
		updateOneagent(t, true)
	})
	t.Run(`only process module config update`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: testVersion,
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("other")
		previousHash := "1"
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)
		updater.fs.MkdirAll(targetDir, 0755)

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			previousHash,
			&processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, "", currentVersion)
	})
	t.Run(`do nothing`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: testVersion,
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		previousHash := "1"
		processModuleCache := createTestProcessModuleConfigCache(previousHash)
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		updater.fs.MkdirAll(targetDir, 0755)

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			previousHash,
			&processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, "", currentVersion)
	})
	t.Run(`failed install`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						Version: testVersion,
					},
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("1")
		previousHash := ""
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		updater.installer.(*installer.InstallerMock).
			On("SetVersion", testVersion).
			Return()
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(fmt.Errorf("BOOM"))

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			previousHash,
			&processModuleCache)

		require.Error(t, err)
		assert.Equal(t, "", currentVersion)
		t_utils.AssertEvents(t,
			updater.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeWarning,
					Reason:    failedInstallAgentVersionEvent,
				},
			},
		)
	})
}

func updateOneagent(t *testing.T, alreadyInstalled bool) {
	dk := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			LatestAgentVersionUnixPaas: testVersion,
		},
	}
	updater := createTestAgentUpdater(t, &dk)
	previousHash := "1"
	processModuleCache := createTestProcessModuleConfigCache(previousHash)
	targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
	installerCalled := false
	updater.installer.(*installer.InstallerMock).
		On("SetVersion", testVersion).
		Return()
	updater.installer.(*installer.InstallerMock).
		On("InstallAgent", targetDir).
		Run(func(args mock.Arguments) {
			installerCalled = true
		}).
		Return(nil)
	updater.installer.(*installer.InstallerMock).
		On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
		Return(nil)
	if alreadyInstalled {
		_ = updater.fs.MkdirAll(targetDir, 0755)
	}

	currentVersion, err := updater.updateAgent(
		"other",
		testTenantUUID,
		previousHash,
		&processModuleCache)

	require.NoError(t, err)
	assert.Equal(t, testVersion, currentVersion)
	assert.Equal(t, !alreadyInstalled, installerCalled)
	t_utils.AssertEvents(t,
		updater.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeNormal,
				Reason:    installAgentVersionEvent,
			},
		},
	)

}

func createTestAgentUpdater(t *testing.T, dk *dynatracev1beta1.DynaKube) *agentUpdater {
	client := dtclient.MockDynatraceClient{}
	path := metadata.PathResolver{RootDir: "test"}
	fs := afero.NewMemMapFs()
	rec := record.NewFakeRecorder(10)

	updater := newAgentUpdater(&client, path, fs, rec, dk)
	require.NotNil(t, updater)
	assert.NotNil(t, updater.installer)

	updater.installer = &installer.InstallerMock{}

	return updater
}

func createTestProcessModuleConfigCache(hash string) processModuleConfigCache {
	return processModuleConfigCache{
		ProcessModuleConfig: &testProcessModuleConfig,
		Hash:                hash,
	}
}
