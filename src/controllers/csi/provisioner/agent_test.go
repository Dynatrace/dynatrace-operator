package csiprovisioner

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testVersion = "test"
)

func TestNewAgentUpdater(t *testing.T) {
	t.Run(`create`, func(t *testing.T) {
		createTestAgentUpdater(t,
			&dynatracev1beta1.DynaKube{
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: "https://" + testTenantUUID + ".dynatrace.com",
				},
			})
	})
}

func TestUpdateAgent(t *testing.T) {
	t.Run(`fresh install`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "https://" + testTenantUUID + ".dynatrace.com",
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						Version: testVersion,
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("1")
		targetDir := updater.targetDir
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(nil)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			&processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)
		t_utils.AssertEvents(t,
			updater.recorder.recorder.(*record.FakeRecorder).Events,
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
				APIURL: "https://" + testTenantUUID + ".dynatrace.com",
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: testVersion,
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("other")
		targetDir := updater.targetDir
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)
		_ = updater.fs.MkdirAll(targetDir, 0755)

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			&processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, "", currentVersion)
	})
	t.Run(`failed install`, func(t *testing.T) {
		dk := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "https://" + testTenantUUID + ".dynatrace.com",
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							Version: testVersion,
						},
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		updater := createTestAgentUpdater(t, &dk)
		processModuleCache := createTestProcessModuleConfigCache("1")
		targetDir := updater.targetDir
		updater.installer.(*installer.InstallerMock).
			On("SetVersion", testVersion).
			Return()
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(fmt.Errorf("BOOM"))

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			&processModuleCache)

		require.Error(t, err)
		assert.Equal(t, "", currentVersion)
		t_utils.AssertEvents(t,
			updater.recorder.recorder.(*record.FakeRecorder).Events,
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
				APIURL:           "https://" + testTenantUUID + ".dynatrace.com",
				CustomPullSecret: pullSecretName,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: image + ":" + tag,
						},
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		mockedPullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte("{}"),
			},
		}
		updater := createTestAgentUpdater(t, &dk, mockedPullSecret)
		targetDir := updater.targetDir
		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(nil)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)

		currentVersion, err := updater.updateAgent(
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
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		mockedObjects := []client.Object{
			&corev1.Secret{
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
			},
		}
		updater := createTestAgentUpdater(t, &dk, mockedObjects...)
		targetDir := updater.targetDir

		updater.installer.(*installer.InstallerMock).
			On("InstallAgent", targetDir).
			Return(nil)
		updater.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", targetDir, &testProcessModuleConfig).
			Return(nil)

		currentVersion, err := updater.updateAgent(
			testVersion,
			testTenantUUID,
			&processModuleConfig)
		require.NoError(t, err)
		assert.Equal(t, tag, currentVersion)
		_, err = updater.fs.Stat(updater.path.ImageCertPath(testTenantUUID))
		assert.Error(t, err)
	})
}

func updateOneagent(t *testing.T, alreadyInstalled bool) {
	dk := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://" + testTenantUUID + ".dynatrace.com",
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			LatestAgentVersionUnixPaas: testVersion,
			ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: tenantUUID,
			},
		},
	}
	updater := createTestAgentUpdater(t, &dk)
	previousHash := "1"
	processModuleCache := createTestProcessModuleConfigCache(previousHash)
	targetDir := updater.targetDir
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
		&processModuleCache)

	require.NoError(t, err)
	if installerCalled {
		assert.Equal(t, testVersion, currentVersion)
	} else {
		assert.Empty(t, currentVersion)
	}

	assert.Equal(t, !alreadyInstalled, installerCalled)
}

func createTestAgentUpdater(t *testing.T, dk *dynatracev1beta1.DynaKube, obj ...client.Object) *agentUpdater {
	mockedClient := dtclient.MockDynatraceClient{}
	path := metadata.PathResolver{RootDir: "test"}
	fs := afero.NewMemMapFs()
	rec := record.NewFakeRecorder(10)

	updater, err := newAgentUpdater(context.TODO(), fs, fake.NewClient(obj...), &mockedClient, path, rec, dk)
	require.NoError(t, err)
	updater.installer = &installer.InstallerMock{}

	return updater
}

func createTestProcessModuleConfigCache(hash string) processModuleConfigCache {
	return processModuleConfigCache{
		ProcessModuleConfig: &testProcessModuleConfig,
		Hash:                hash,
	}
}
