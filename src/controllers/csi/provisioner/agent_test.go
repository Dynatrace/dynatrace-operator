package csiprovisioner

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	t_utils "github.com/Dynatrace/dynatrace-operator/src/testing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

const (
	testVersion = "test"
)

func TestNewAgentUpdater(t *testing.T) {
	t.Run(`create`, func(t *testing.T) {
		createTestAgentUpdater(t, &dynatracev1beta1.DynaKube{})
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
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							Version: testVersion,
						},
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
	ctx := context.TODO()
	t.Run(`fresh install`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							Version: testVersion,
						},
					},
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("1")
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
			ctx,
			testVersion,
			testTenantUUID,
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
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)
		_ = updater.fs.MkdirAll(targetDir, 0755)

		currentVersion, err := updater.updateAgent(
			ctx,
			testVersion,
			testTenantUUID,
			&processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, "", currentVersion)
	})
	t.Run(`failed install`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							Version: testVersion,
						},
					},
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("1")
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, testVersion)
		updater.installer.(*installer.InstallerMock).
			On("SetVersion", testVersion).
			Return()
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(fmt.Errorf("BOOM"))

		currentVersion, err := updater.updateAgent(
			ctx,
			testVersion,
			testTenantUUID,
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
	t.Run(`codeModulesImage set`, func(t *testing.T) {
		image := "test-image"
		tag := "tag"
		pullSecretName := "test-pull-secret"
		testNamespace := "test-namespace"
		processModuleConfig := createTestProcessModuleConfigCache("1")
		dk := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				CustomPullSecret: pullSecretName,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: image + ":" + tag,
						},
					},
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, tag)
		updater.apiReader = fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte("{}"),
			},
		})
		updater.installer.(*installer.InstallerMock).
			On("SetImageInfo", mock.Anything).
			Return()
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(nil)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)

		currentVersion, err := updater.updateAgent(
			ctx,
			testVersion,
			testTenantUUID,
			&processModuleConfig)
		require.NoError(t, err)
		assert.Equal(t, tag, currentVersion)
	})
	t.Run(`codeModulesImage + trustedCA set`, func(t *testing.T) {
		image := "test-image"
		tag := "tag"
		pullSecretName := "test-pull-secret"
		trustedCAName := "test-trusted-ca"
		testNamespace := "test-namespace"
		processModuleConfig := createTestProcessModuleConfigCache("1")
		dk := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				CustomPullSecret: pullSecretName,
				TrustedCAs:       trustedCAName,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: image + ":" + tag,
						},
					},
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		targetDir := updater.path.AgentBinaryDirForVersion(testTenantUUID, tag)
		updater.apiReader = fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte("{}"),
			},
		},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      trustedCAName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					dtclient.CustomCertificatesConfigMapKey: "I-am-a-cert-trust-me",
				},
			})
		expectedDockerConfig := dockerconfig.DockerConfig{
			Dynakube:         &dk,
			ApiReader:        updater.apiReader,
			TrustedCertsPath: filepath.Join(targetDir, "ca.crt"),
		}

		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(nil)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)

		currentVersion, err := updater.updateAgent(
			ctx,
			testVersion,
			testTenantUUID,
			&processModuleConfig)
		require.NoError(t, err)
		assert.Equal(t, tag, currentVersion)
		_, err = updater.fs.Stat(expectedDockerConfig.TrustedCertsPath)
		assert.Error(t, err)
	})
}

func updateOneagent(t *testing.T, alreadyInstalled bool) {
	ctx := context.TODO()
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
		ctx,
		"other",
		testTenantUUID,
		&processModuleCache)

	require.NoError(t, err)
	if installerCalled {
		assert.Equal(t, testVersion, currentVersion)
	} else {
		assert.Empty(t, currentVersion)
	}

	assert.Equal(t, !alreadyInstalled, installerCalled)
}

func createTestAgentUpdater(t *testing.T, dk *dynatracev1beta1.DynaKube) *agentUpdater {
	client := dtclient.MockDynatraceClient{}
	path := metadata.PathResolver{RootDir: "test"}
	fs := afero.NewMemMapFs()
	rec := record.NewFakeRecorder(10)

	updater := newAgentUpdater(fake.NewClient(), &client, path, fs, rec, dk)
	updater.installer = &installer.InstallerMock{}
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
