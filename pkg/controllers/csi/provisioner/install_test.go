package csiprovisioner

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry/mocks"
	t_utils "github.com/Dynatrace/dynatrace-operator/pkg/util/testing"
	mockedclient "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	mockedinstaller "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	"github.com/opencontainers/go-digest"
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
	testTenantUUID = "zib123"
)

func TestUpdateAgent(t *testing.T) {
	testVersion := "test"
	testImageDigest := "7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	t.Run("zip install", func(t *testing.T) {
		dk := createTestDynaKubeWithZip(testVersion)
		provisioner := createTestProvisioner()
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.CodeModulesVersion())
		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testVersion))
		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentZip(dk, mockedclient.NewClient(t), processModule)
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
		processModule := createTestProcessModuleConfig(revision)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", newTargetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, newVersion))
		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentZip(dk, mockedclient.NewClient(t), processModule)
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
		processModule := createTestProcessModuleConfig(revision)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", targetDir).
			Return(false, nil)

		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)
		currentVersion, err := provisioner.installAgentZip(dk, mockedclient.NewClient(t), processModule)

		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)
	})
	t.Run("failed install", func(t *testing.T) {
		dockerconfigjsonContent := `{"auths":{}}`
		dk := createTestDynaKubeWithImage(testImageDigest)
		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", targetDir).
			Return(false, fmt.Errorf("BOOM"))
		mockRegistryClient(provisioner, "sha256:"+testImageDigest)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(dk, processModule)

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
		processModule := createTestProcessModuleConfig(revision)

		dk := createTestDynaKubeWithImage(testImageDigest)
		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testImageDigest))
		mockRegistryClient(provisioner, "sha256:"+testImageDigest)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(dk, processModule)
		require.NoError(t, err)
		assert.Equal(t, testImageDigest, currentVersion)
	})
	t.Run("codeModulesImage set with custom pull secret", func(t *testing.T) {
		pullSecretName := "test-pull-secret"
		dockerconfigjsonContent := `{"auths":{}}`
		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)

		dk := createTestDynaKubeWithImage(testImageDigest)
		dk.Spec.CustomPullSecret = pullSecretName

		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testImageDigest))
		mockRegistryClient(provisioner, "sha256:"+testImageDigest)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(dk, processModule)
		require.NoError(t, err)
		assert.Equal(t, testImageDigest, currentVersion)
	})
	t.Run("codeModulesImage + trustedCA set", func(t *testing.T) {
		pullSecretName := "test-pull-secret"
		trustedCAName := "test-trusted-ca"
		customCertContent := `
-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUdKGNuWxm1t7auCtk+RYAgMKC4wkwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMzA4MDcxMzUzMjBaFw0yNDA4
MDYxMzUzMjBaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDGkW280WZbTyHPiNQXHVaWW/C3ZbaKh5cuarQUkHZc
1SVfFELuJXm3YAA5ZOtwaIuqsSO9Yieao0kWYWWCSyFdwcOIl5H85n9YaZ1/8ki3
af7TwH1UppA3Zh24eV9ME+uJKsmn4AkMVaM9EKUaOTybZD6Sc0jxsmec9yDuE4md
P0vqIshcd6VmxruPnzzmOEXggP3QPFF5s9017uPnQ7k2kU8b0MG19HS2opeeSO59
R2+kg/Xkz8UnCa5y+OSORW20DHjwc7DUr/Gr78X49iiFBzBewBfeqxQKwtYcC9eB
DxiDWiXENUnsS0EkMs4jNFjgiAJTzx6rBa4xiwe7SJWfAgMBAAGjUzBRMB0GA1Ud
DgQWBBR+L23VHT1LLmpAwz4esbVmfSCOdDAfBgNVHSMEGDAWgBR+L23VHT1LLmpA
wz4esbVmfSCOdDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCj
jU/luq9dNiZi6fhgfhDQRuZEYnHSV8L3+hEDn1j6Gn02c9wNcCDjOBH4i8pJz8g2
x+Z1SNALXFcr+bGJQx94lw7S1Vm84YxELyNbwVYuHo+7aLAUXSQ62RMIhEJ/NCzW
yN0j8PhweOTBwUtvzPa+71f1gNbDgkfXqgLSXBgjNvolcg/lefmKBs0pU8swOmX1
q8nrWV12953Gf9sMJ0mFP5/Lcv4l1SdnFLOSdVjWF4RX+SjnVgiHSuJxp9k3QiXz
5dlfTqc9/qZa1PRq4hdq/3Rs42Hiwa3FTWSgqjM1qcDycQtTIAeZu2zfYDQDkYcI
NK85cEJwyxQ+wahdNGUD
-----END CERTIFICATE-----
`
		dockerconfigjsonContent := `{"auths":{}}`
		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)

		dk := createTestDynaKubeWithImage(testImageDigest)
		dk.Spec.CustomPullSecret = pullSecretName
		dk.Spec.TrustedCAs = trustedCAName

		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent), createMockedCAConfigMap(dk, customCertContent))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(testImageDigest)
		installerMock := mockedinstaller.NewInstaller(t)
		installerMock.
			On("InstallAgent", targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testImageDigest))
		mockRegistryClient(provisioner, "sha256:"+testImageDigest)
		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(dk, processModule)
		require.NoError(t, err)
		assert.Equal(t, testImageDigest, currentVersion)
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

func createMockedPullSecret(dynakube dynatracev1beta1.DynaKube, pullSecretContent string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.PullSecretName(),
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

func mockRegistryClient(provisioner *OneAgentProvisioner, imageDigest string) {
	provisioner.registryClientBuilder = func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		regMock := &mocks.MockImageGetter{}
		regMock.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Digest: digest.Digest(imageDigest)}, nil)
		return regMock, nil
	}
}

func mockImageInstallerBuilder(mock *mockedinstaller.Installer) imageInstallerBuilder {
	return func(f afero.Fs, p *image.Properties) (installer.Installer, error) {
		return mock, nil
	}
}

func mockUrlInstallerBuilder(mock *mockedinstaller.Installer) urlInstallerBuilder {
	return func(f afero.Fs, c dtclient.Client, p *url.Properties) installer.Installer {
		return mock
	}
}

func createTestProcessModuleConfig(revision uint) *dtclient.ProcessModuleConfig {
	return &dtclient.ProcessModuleConfig{
		Revision: revision,
		Properties: []dtclient.ProcessModuleProperty{
			{
				Section: "test",
				Key:     "test",
				Value:   "test3",
			},
		},
	}
}
