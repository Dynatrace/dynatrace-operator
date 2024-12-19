package csiprovisioner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("no dynakube(ie.: delete case) => do nothing, no error", func(t *testing.T) { // TODO: Replace "do nothing" with "run GC"
		prov := createProvisioner(t)
		dk := createDynaKubeBase(t)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.False(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube doesn't need app-injection => only setup base fs, no error, long requeue", func(t *testing.T) {
		dk := createDynaKubeNoCSI(t)
		prov := createProvisioner(t, dk)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, longRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube status not ready => only setup base fs, no error, short requeue", func(t *testing.T) {
		dk := createNotReadyDynaKube(t)
		prov := createProvisioner(t, dk)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, shortRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with version => url installer used, no error", func(t *testing.T) {
		dk := createDynaKubeWithVersion(t)
		prov := createProvisioner(t, dk, createToken(t, dk), createPMCSecret(t, dk))
		installer := createSuccessfulInstaller(t)
		prov.urlInstallerBuilder = mockUrlInstallerBuilder(t, installer)
		prov.dynatraceClientBuilder = mockSuccessfulDtClientBuilder(t)
		createPMCSourceFile(t, prov, dk)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
		installer.AssertCalled(t, "InstallAgent", mock.Anything, mock.Anything)
	})

	t.Run("dynakube with version, issue with dtc => fail before installer creation", func(t *testing.T) {
		dk := createDynaKubeWithVersion(t)
		prov := createProvisioner(t, dk, createToken(t, dk))
		prov.dynatraceClientBuilder = mockFailingDtClientBuilder(t)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.Error(t, err)
		require.NotNil(t, result)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with image => image installer used, dtclient not created, no error", func(t *testing.T) {
		dk := createDynaKubeWithImage(t)
		prov := createProvisioner(t, dk, createPMCSecret(t, dk))
		installer := createSuccessfulInstaller(t)
		prov.imageInstallerBuilder = mockImageInstallerBuilder(t, installer)
		createPMCSourceFile(t, prov, dk)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
		installer.AssertCalled(t, "InstallAgent", mock.Anything, mock.Anything)
	})

	t.Run("installer fails => error", func(t *testing.T) {
		dk := createDynaKubeWithImage(t)
		prov := createProvisioner(t, dk)
		installer := createFailingInstaller(t)
		prov.imageInstallerBuilder = mockImageInstallerBuilder(t, installer)

		result, err := prov.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.Error(t, err)
		require.NotNil(t, result)

		assert.True(t, areFsDirsCreated(t, prov, dk))
		installer.AssertCalled(t, "InstallAgent", mock.Anything, mock.Anything)
	})
}

func areFsDirsCreated(t *testing.T, prov OneAgentProvisioner, dk *dynakube.DynaKube) bool {
	t.Helper()

	neededFolders := []string{
		prov.path.DynaKubeDir(dk.GetName()),
		prov.path.AgentSharedBinaryDirBase(),
	}
	for _, folder := range neededFolders {
		stat, err := prov.fs.Stat(folder)
		if err != nil || stat == nil || !stat.IsDir() {
			return false
		}
	}

	return true
}

func createProvisioner(t *testing.T, objs ...client.Object) OneAgentProvisioner {
	t.Helper()

	return OneAgentProvisioner{
		fs:        afero.NewMemMapFs(),
		path:      metadata.PathResolver{},
		apiReader: fake.NewClient(objs...),
	}
}

func createDynaKubeWithVersion(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	version := "test-version"
	dk.Spec.OneAgent = dynakube.OneAgentSpec{
		ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
			Version: version,
		},
	}
	dk.Status.CodeModules.Version = version

	return dk
}

func createDynaKubeWithImage(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	imageId := "test-image"
	dk.Spec.OneAgent = dynakube.OneAgentSpec{
		CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
			AppInjectionSpec: dynakube.AppInjectionSpec{CodeModulesImage: imageId},
		},
	}
	dk.Status.CodeModules.ImageID = imageId

	return dk
}

func createNotReadyDynaKube(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	dk.Spec.OneAgent = dynakube.OneAgentSpec{
		CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
	}

	return dk
}

func createDynaKubeNoCSI(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	dk.Spec.OneAgent = dynakube.OneAgentSpec{
		ClassicFullStack: &dynakube.HostInjectSpec{},
	}

	return dk
}

func createDynaKubeBase(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "test-ns",
		},
	}
}

func createSuccessfulInstaller(t *testing.T) *installermock.Installer {
	t.Helper()

	m := installermock.NewInstaller(t)
	m.On("InstallAgent", mock.Anything, mock.Anything).Return(true, nil)

	return m
}

func createFailingInstaller(t *testing.T) *installermock.Installer {
	t.Helper()

	m := installermock.NewInstaller(t)
	m.On("InstallAgent", mock.Anything, mock.Anything).Return(false, errors.New("BOOM"))

	return m
}

func mockUrlInstallerBuilder(t *testing.T, mockedInstaller *installermock.Installer) urlInstallerBuilder {
	t.Helper()

	return func(f afero.Fs, _ dtclient.Client, _ *url.Properties) installer.Installer {
		return mockedInstaller
	}
}

func mockImageInstallerBuilder(t *testing.T, mockedInstaller *installermock.Installer) imageInstallerBuilder {
	t.Helper()

	return func(_ context.Context, _ afero.Fs, _ *image.Properties) (installer.Installer, error) {
		return mockedInstaller, nil
	}
}

func createToken(t *testing.T, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.Tokens(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			dtclient.ApiToken: []byte("this is a token"),
		},
	}
}

func createPMCSecret(t *testing.T, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.GetName() + processmoduleconfigsecret.SecretSuffix,
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			processmoduleconfigsecret.SecretKeyProcessModuleConfig: getPMC(t),
		},
	}
}

func createPMCSourceFile(t *testing.T, prov OneAgentProvisioner, dk *dynakube.DynaKube) {
	t.Helper()

	pmcPath := filepath.Join(prov.getTargetDir(*dk), processmoduleconfig.RuxitAgentProcPath)
	pmcDir := filepath.Dir(pmcPath)
	require.NoError(t, prov.fs.MkdirAll(pmcDir, os.ModePerm))

	pmcFile, err := prov.fs.OpenFile(pmcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	require.NoError(t, err)
	_, err = pmcFile.Write(getPMC(t))
	require.NoError(t, err)
}

func getPMC(t *testing.T) []byte {
	t.Helper()

	pmc := dtclient.ProcessModuleConfig{
		Revision: 0,
		Properties: []dtclient.ProcessModuleProperty{
			{Section: "test-section", Key: "test-key", Value: "test-value"},
		},
	}

	pmcJson, err := json.Marshal(pmc)
	require.NoError(t, err)

	return pmcJson
}

func mockFailingDtClientBuilder(t *testing.T) dynatraceclient.Builder {
	t.Helper()

	mockDtcBuilder := dtbuildermock.NewBuilder(t)
	mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("Build", mock.Anything).Return(nil, errors.New("BOOM"))

	return mockDtcBuilder
}

func mockSuccessfulDtClientBuilder(t *testing.T) dynatraceclient.Builder {
	t.Helper()

	mockDtcBuilder := dtbuildermock.NewBuilder(t)
	mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.On("Build", mock.Anything).Return(dtclientmock.NewClient(t), nil)

	return mockDtcBuilder
}
