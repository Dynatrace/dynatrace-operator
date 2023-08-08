package csiprovisioner

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/image"
	"github.com/Dynatrace/dynatrace-operator/src/installer/url"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
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

func TestUpdateAgent(t *testing.T) {
	testVersion := "test"
	testImageDigest := "7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	t.Run("zip install", func(t *testing.T) {
		dk := createTestDynaKubeWithZip(testVersion)
		provisioner := createTestProvisioner()
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.CodeModulesVersion())
		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testVersion))
		installerMock.
			On("Cleanup").
			Return(nil)
		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentZip(dk, &dtclient.MockDynatraceClient{}, &processModuleCache)
		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)
		t_utils.AssertEvents(t,
			provisioner.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    installAgentVersionEvent,
				},
			},
		)
	})
	t.Run("zip update", func(t *testing.T) {
		dk := createTestDynaKubeWithZip(testVersion)
		provisioner := createTestProvisioner()
		previousTargetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.CodeModulesVersion())
		previousSourceConfigPath := filepath.Join(previousTargetDir, processmoduleconfig.RuxitAgentProcPath)
		_ = provisioner.fs.MkdirAll(previousTargetDir, 0755)
		_, _ = provisioner.fs.Create(previousSourceConfigPath)

		newVersion := "new"
		dk.Status.CodeModules.Version = newVersion
		newTargetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.CodeModulesVersion())

		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", newTargetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, newVersion))
		installerMock.
			On("Cleanup").
			Return(nil)
		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentZip(dk, &dtclient.MockDynatraceClient{}, &processModuleCache)
		require.NoError(t, err)
		assert.Equal(t, newVersion, currentVersion)
	})
	t.Run("only process module config update", func(t *testing.T) {
		dk := createTestDynaKubeWithZip(testVersion)
		provisioner := createTestProvisioner()
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.CodeModulesVersion())
		sourceConfigPath := filepath.Join(targetDir, processmoduleconfig.RuxitAgentProcPath)
		_ = provisioner.fs.MkdirAll(targetDir, 0755)
		_, _ = provisioner.fs.Create(sourceConfigPath)
		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", targetDir).
			Return(false, nil)
		installerMock.
			On("Cleanup").
			Return(nil)

		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)
		currentVersion, err := provisioner.installAgentZip(dk, &dtclient.MockDynatraceClient{}, &processModuleCache)

		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)
	})
	t.Run("failed install", func(t *testing.T) {
		dockerconfigjsonContent := `{"auths":{}}`
		dk := createTestDynaKubeWithImage(testImageDigest)
		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", targetDir).
			Return(false, fmt.Errorf("BOOM"))
		installerMock.
			On("Cleanup").
			Return(nil)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(context.TODO(), dk, &processModuleCache)

		require.Error(t, err)
		assert.Equal(t, "", currentVersion)
		t_utils.AssertEvents(t,
			provisioner.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeWarning,
					Reason:    failedInstallAgentVersionEvent,
				},
			},
		)
	})
	t.Run("codeModulesImage set without custom pull secret", func(t *testing.T) {
		dockerconfigjsonContent := `{"auths":{}}`
		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)

		dk := createTestDynaKubeWithImage(testImageDigest)
		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testImageDigest))
		installerMock.
			On("Cleanup").
			Return(nil)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(context.TODO(), dk, &processModuleCache)
		require.NoError(t, err)
		assert.Equal(t, testImageDigest, currentVersion)

		dockerJsonPath := path.Join(dockerconfig.TmpPath, dockerconfig.RegistryAuthDir, dk.Name)
		checkFilesCreated(t, provisioner.fs, dockerJsonPath, dockerconfigjsonContent)
	})
	t.Run("codeModulesImage set with custom pull secret", func(t *testing.T) {
		pullSecretName := "test-pull-secret"
		dockerconfigjsonContent := `{"auths":{}}`
		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)

		dk := createTestDynaKubeWithImage(testImageDigest)
		dk.Spec.CustomPullSecret = pullSecretName

		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testImageDigest))
		installerMock.
			On("Cleanup").
			Return(nil)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(context.TODO(), dk, &processModuleCache)
		require.NoError(t, err)
		assert.Equal(t, testImageDigest, currentVersion)

		dockerJsonPath := path.Join(dockerconfig.TmpPath, dockerconfig.RegistryAuthDir, dk.Name)
		checkFilesCreated(t, provisioner.fs, dockerJsonPath, dockerconfigjsonContent)
	})
	t.Run("codeModulesImage + trustedCA set", func(t *testing.T) {
		pullSecretName := "test-pull-secret"
		trustedCAName := "test-trusted-ca"
		customCertContent := "I-am-a-cert-trust-me"
		dockerconfigjsonContent := `{"auths":{}}`
		var revision uint = 3
		processModuleCache := createTestProcessModuleConfigCache(revision)

		dk := createTestDynaKubeWithImage(testImageDigest)
		dk.Spec.CustomPullSecret = pullSecretName
		dk.Spec.TrustedCAs = trustedCAName

		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent), createMockedCAConfigMap(dk, customCertContent))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := &installer.Mock{}
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testImageDigest))
		installerMock.
			On("Cleanup").
			Return(nil)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(context.TODO(), dk, &processModuleCache)
		require.NoError(t, err)
		assert.Equal(t, testImageDigest, currentVersion)

		dockerJsonPath := path.Join(dockerconfig.TmpPath, dockerconfig.RegistryAuthDir, dk.Name)
		checkFilesCreated(t, provisioner.fs, dockerJsonPath, dockerconfigjsonContent)

		caFilePath := path.Join(dockerconfig.TmpPath, dockerconfig.CADir, dk.Name, dockerconfig.TrustedCertFileName)
		checkFilesCreated(t, provisioner.fs, caFilePath, customCertContent)
	})
}

func mockFsAfterInstall(provisioner *OneAgentProvisioner, version string) func(mock.Arguments) {
	return func(mock.Arguments) {
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(version)
		sourceConfigPath := filepath.Join(targetDir, processmoduleconfig.RuxitAgentProcPath)
		_ = provisioner.fs.MkdirAll(targetDir, 0755)
		_ = provisioner.fs.MkdirAll(filepath.Dir(sourceConfigPath), 0755)
		_, _ = provisioner.fs.Create(sourceConfigPath)
	}
}

func checkFilesCreated(t *testing.T, fs afero.Fs, caFilePath string, certContent string) {
	caFile, err := fs.Open(caFilePath)
	require.NoError(t, err)

	caFileContent, err := io.ReadAll(caFile)
	require.NoError(t, err)
	require.Equal(t, certContent, string(caFileContent))
}

func createMockedPullSecret(dynakube dynatracev1beta1.DynaKube, pullSecretContent string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.PullSecret(),
			Namespace: dynakube.Namespace,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(pullSecretContent),
		},
	}
}

func createMockedCAConfigMap(dynakube dynatracev1beta1.DynaKube, certContent string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.Spec.TrustedCAs,
			Namespace: dynakube.Namespace,
		},
		Data: map[string]string{
			dynatracev1beta1.TrustedCAKey: certContent,
		},
	}
}

func createTestDynaKubeWithImage(imageDigest string) dynatracev1beta1.DynaKube {
	imageID := "some.registry.com/image:1.234.345@sha256:" + imageDigest
	return dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-ns",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://" + testTenantUUID + ".dynatrace.com",
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			CodeModules: dynatracev1beta1.CodeModulesStatus{
				VersionStatus: status.VersionStatus{
					ImageID: imageID,
				},
			},
		},
	}
}

func createTestDynaKubeWithZip(version string) dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-ns",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://" + testTenantUUID + ".dynatrace.com",
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			CodeModules: dynatracev1beta1.CodeModulesStatus{
				VersionStatus: status.VersionStatus{
					Version: version,
				},
			},
		},
	}
}

func createTestProvisioner(obj ...client.Object) *OneAgentProvisioner {
	path := metadata.PathResolver{RootDir: "test"}
	fs := afero.NewMemMapFs()
	rec := record.NewFakeRecorder(10)
	db := metadata.FakeMemoryDB()

	fakeClient := fake.NewClient(obj...)
	provisioner := &OneAgentProvisioner{
		client:    fakeClient,
		apiReader: fakeClient,
		fs:        fs,
		recorder:  rec,
		path:      path,
		db:        db,
	}

	return provisioner
}

func mockImageInstallerBuilder(mock *installer.Mock) imageInstallerBuilder {
	return func(f afero.Fs, p *image.Properties) installer.Installer {
		return mock
	}
}

func mockUrlInstallerBuilder(mock *installer.Mock) urlInstallerBuilder {
	return func(f afero.Fs, c dtclient.Client, p *url.Properties) installer.Installer {
		return mock
	}
}
