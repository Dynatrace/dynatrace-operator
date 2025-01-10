package csiprovisioner

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	t_utils "github.com/Dynatrace/dynatrace-operator/pkg/util/testing"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
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
	ctx := context.Background()
	testVersion := "test"
	testImage := "my-image/1223:123"

	t.Run("zip install", func(t *testing.T) {
		dk := createTestDynaKubeWithZip(testVersion)
		provisioner := createTestProvisioner()
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.OneAgent().GetCodeModulesVersion())

		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, testVersion))

		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentZip(ctx, dk, dtclientmock.NewClient(t), processModule)
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
		previousTargetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.OneAgent().GetCodeModulesVersion())
		previousSourceConfigPath := filepath.Join(previousTargetDir, processmoduleconfig.RuxitAgentProcPath)
		_ = provisioner.fs.MkdirAll(previousTargetDir, 0755)
		_, _ = provisioner.fs.Create(previousSourceConfigPath)

		newVersion := "new"
		dk.Status.CodeModules.Version = newVersion
		newTargetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.OneAgent().GetCodeModulesVersion())

		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), newTargetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, newVersion))

		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentZip(ctx, dk, dtclientmock.NewClient(t), processModule)
		require.NoError(t, err)
		assert.Equal(t, newVersion, currentVersion)
	})
	t.Run("only process module config update", func(t *testing.T) {
		dk := createTestDynaKubeWithZip(testVersion)
		provisioner := createTestProvisioner()
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(dk.OneAgent().GetCodeModulesVersion())
		sourceConfigPath := filepath.Join(targetDir, processmoduleconfig.RuxitAgentProcPath)
		_ = provisioner.fs.MkdirAll(targetDir, 0755)
		_, _ = provisioner.fs.Create(sourceConfigPath)

		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(false, nil)

		provisioner.urlInstallerBuilder = mockUrlInstallerBuilder(installerMock)
		currentVersion, err := provisioner.installAgentZip(ctx, dk, dtclientmock.NewClient(t), processModule)

		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)
	})
	t.Run("failed install", func(t *testing.T) {
		dockerconfigjsonContent := `{"auths":{}}`
		dk := createTestDynaKubeWithImage(testImage)
		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))

		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64.StdEncoding.EncodeToString([]byte(testImage)))
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(false, fmt.Errorf("BOOM"))

		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(ctx, dk, processModule)

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

		dk := createTestDynaKubeWithImage(testImage)
		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		base64Image := base64.StdEncoding.EncodeToString([]byte(testImage))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64Image)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, base64Image))

		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(ctx, dk, processModule)
		require.NoError(t, err)
		assert.Equal(t, base64Image, currentVersion)
	})
	t.Run("codeModulesImage set with custom pull secret", func(t *testing.T) {
		pullSecretName := "test-pull-secret"
		dockerconfigjsonContent := `{"auths":{}}`

		var revision uint = 3
		processModule := createTestProcessModuleConfig(revision)

		dk := createTestDynaKubeWithImage(testImage)
		dk.Spec.CustomPullSecret = pullSecretName

		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent))
		base64Image := base64.StdEncoding.EncodeToString([]byte(testImage))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64Image)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, base64Image))

		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(ctx, dk, processModule)
		require.NoError(t, err)
		assert.Equal(t, base64Image, currentVersion)
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

		dk := createTestDynaKubeWithImage(testImage)
		dk.Spec.CustomPullSecret = pullSecretName
		dk.Spec.TrustedCAs = trustedCAName

		provisioner := createTestProvisioner(createMockedPullSecret(dk, dockerconfigjsonContent), createMockedCAConfigMap(dk, customCertContent))
		base64Image := base64.StdEncoding.EncodeToString([]byte(testImage))
		targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64Image)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(true, nil).Run(mockFsAfterInstall(provisioner, base64Image))

		provisioner.imageInstallerBuilder = mockImageInstallerBuilder(installerMock)

		currentVersion, err := provisioner.installAgentImage(ctx, dk, processModule)
		require.NoError(t, err)
		assert.Equal(t, base64Image, currentVersion)
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

func createMockedPullSecret(dk dynakube.DynaKube, pullSecretContent string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.PullSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(pullSecretContent),
		},
	}
}

func createMockedCAConfigMap(dk dynakube.DynaKube, certContent string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.Spec.TrustedCAs,
			Namespace: dk.Namespace,
		},
		Data: map[string]string{
			dynakube.TrustedCAKey: certContent,
		},
	}
}

func createTestDynaKubeWithImage(image string) dynakube.DynaKube {
	return *addFakeTenantUUID(&dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-ns",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://" + testTenantUUID + ".dynatrace.com",
		},
		Status: dynakube.DynaKubeStatus{
			CodeModules: oneagent.CodeModulesStatus{
				VersionStatus: status.VersionStatus{
					ImageID: image,
				},
			},
		},
	})
}

func createTestDynaKubeWithZip(version string) dynakube.DynaKube {
	return *addFakeTenantUUID(&dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-ns",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://" + testTenantUUID + ".dynatrace.com",
		},
		Status: dynakube.DynaKubeStatus{
			CodeModules: oneagent.CodeModulesStatus{
				VersionStatus: status.VersionStatus{
					Version: version,
				},
			},
		},
	})
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

func mockImageInstallerBuilder(mock *installermock.Installer) imageInstallerBuilder {
	return func(_ context.Context, _ afero.Fs, _ *image.Properties) (installer.Installer, error) {
		return mock, nil
	}
}

func mockUrlInstallerBuilder(mock *installermock.Installer) urlInstallerBuilder {
	return func(_ afero.Fs, _ dtclient.Client, _ *url.Properties) installer.Installer {
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
