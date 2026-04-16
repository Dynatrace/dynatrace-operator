package csiprovisioner

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/provisioner/cleanup"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/job"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	t.Run("no dynakube(ie.: delete case) => do nothing, no error", func(t *testing.T) { // TODO: Replace "do nothing" with "run GC"
		prov := createProvisioner(t)
		dk := createDynaKubeBase(t)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.False(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube doesn't need app-injection => no error, long requeue", func(t *testing.T) {
		dk := createDynaKubeNoCSI(t)
		prov := createProvisioner(t, dk)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, longRequeueDuration, result.RequeueAfter)

		assert.False(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("migration mode (csiDriver=false in modules.json) => cleanup only, no install, long requeue", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		dk := createDynaKubeWithImage(t)
		prov := createProvisioner(t, dk)
		// imageInstallerBuilder is intentionally NOT set — if it were called the test would panic
		prov.imageInstallerBuilder = func(_ context.Context, _ *image.Properties) (installer.Installer, error) {
			return nil, nil
		}

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, longRequeueDuration, result.RequeueAfter)

		assert.False(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube status not ready => only setup base fs, no error, short requeue", func(t *testing.T) {
		dk := createNotReadyDynaKube(t)
		prov := createProvisioner(t, dk)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, shortRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube status not ready, status has version, but images would be needed => only setup base fs, no error, short requeue", func(t *testing.T) {
		dk := createDynaKubeWithImage(t)
		dk.Status.CodeModules.Version = dk.Status.CodeModules.ImageID
		dk.Status.CodeModules.ImageID = ""
		prov := createProvisioner(t, dk)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, shortRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube status not ready, status needs version, but still set to `custom-image` => only setup base fs, no error, short requeue", func(t *testing.T) {
		dk := createDynaKubeWithVersion(t)
		dk.Status.CodeModules.Version = string(status.CustomImageVersionSource)
		prov := createProvisioner(t, dk)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, shortRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with version => url installer used, no error", func(t *testing.T) {
		dk := createDynaKubeWithVersion(t)
		prov := createProvisioner(t, dk, createToken(t, dk))
		prov.urlInstallerBuilder = mockURLInstallerBuilder(t, createSuccessfulInstaller(t))
		prov.dynatraceClientBuilder = mockSuccessfulDTClientBuilder(t)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with version, unknown issue with dtc => fail before installer creation", func(t *testing.T) {
		dk := createDynaKubeWithVersion(t)
		prov := createProvisioner(t, dk, createToken(t, dk))
		prov.dynatraceClientBuilder = mockFailingDTClientBuilder(t)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.Error(t, err)
		require.NotNil(t, result)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with version, known issue with dtc => no error, just short requeue", func(t *testing.T) {
		dk := createDynaKubeWithVersion(t)
		prov := createProvisioner(t, dk, createToken(t, dk))

		unavailableInstaller := installermock.NewInstaller(t)
		unavailableInstaller.EXPECT().InstallAgent(mock.Anything, mock.Anything).Return(false, dtclient.ServerError{Code: http.StatusServiceUnavailable})
		prov.urlInstallerBuilder = mockURLInstallerBuilder(t, unavailableInstaller)
		prov.dynatraceClientBuilder = mockSuccessfulDTClientBuilder(t)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, shortRequeueDuration, result.RequeueAfter)
	})

	t.Run("dynakube with image => image installer used, dtclient not created, no error", func(t *testing.T) {
		dk := createDynaKubeWithImage(t)
		prov := createProvisioner(t, dk)
		prov.imageInstallerBuilder = mockImageInstallerBuilder(t, createSuccessfulInstaller(t))

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with job => job installer used, dtclient not created, no error", func(t *testing.T) {
		dk := createDynaKubeWithJobFF(t)
		prov := createProvisioner(t, dk)
		prov.jobInstallerBuilder = mockJobInstallerBuilder(t, createSuccessfulInstaller(t))

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with job + custom-pull-secret => job installer used, dtclient not created, no error", func(t *testing.T) {
		dk := createDynaKubeWithJobFF(t)
		dk.Spec.CustomPullSecret = "test-ps"
		prov := createProvisioner(t, dk)
		prov.jobInstallerBuilder = mockJobInstallerBuilder(t, createSuccessfulInstaller(t), dk.Spec.CustomPullSecret)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with job + helm pull secret env var => helm pull secret included", func(t *testing.T) {
		helmPullSecret := "helm-pull-secret"
		t.Setenv("DT_OPERATOR_PULL_SECRET", helmPullSecret)
		dk := createDynaKubeWithJobFF(t)
		prov := createProvisioner(t, dk)
		prov.jobInstallerBuilder = mockJobInstallerBuilder(t, createSuccessfulInstaller(t), helmPullSecret)

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with job => job installer used, back-off when not ready, no error", func(t *testing.T) {
		dk := createDynaKubeWithJobFF(t)
		prov := createProvisioner(t, dk)
		prov.jobInstallerBuilder = mockJobInstallerBuilder(t, createNotReadyInstaller(t))

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, notReadyRequeueDuration, result.RequeueAfter)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("dynakube with job => job installer used, with error", func(t *testing.T) {
		dk := createDynaKubeWithJobFF(t)
		prov := createProvisioner(t, dk)
		prov.jobInstallerBuilder = mockJobInstallerBuilder(t, createFailingInstaller(t))

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.Error(t, err)
		require.NotNil(t, result)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})

	t.Run("installer fails => error", func(t *testing.T) {
		dk := createDynaKubeWithImage(t)
		prov := createProvisioner(t, dk)
		prov.imageInstallerBuilder = mockImageInstallerBuilder(t, createFailingInstaller(t))

		result, err := prov.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(dk)})
		require.Error(t, err)
		require.NotNil(t, result)

		assert.True(t, areFsDirsCreated(t, prov, dk))
	})
}

func areFsDirsCreated(t *testing.T, prov OneAgentProvisioner, dk *dynakube.DynaKube) bool {
	t.Helper()

	neededFolders := []string{
		prov.path.DynaKubeDir(dk.GetName()),
		prov.path.AgentSharedBinaryDirBase(),
	}
	for _, folder := range neededFolders {
		stat, err := os.Stat(folder)
		if err != nil || stat == nil || !stat.IsDir() {
			return false
		}
	}

	return true
}

func createProvisioner(t *testing.T, objs ...client.Object) OneAgentProvisioner {
	t.Helper()

	path := metadata.PathResolver{RootDir: t.TempDir()}
	apiReader := fake.NewClient(objs...)

	return OneAgentProvisioner{
		path:      path,
		apiReader: apiReader,
		cleaner:   cleanup.New(apiReader, path, mount.NewFakeMounter(nil)),
	}
}

func createDynaKubeWithVersion(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	version := "test-version"
	dk.Spec.OneAgent = oneagent.Spec{
		ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
			Version: version, //nolint:staticcheck
		},
	}
	dk.Status.CodeModules.Version = version

	return dk
}

func createDynaKubeWithImage(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	imageID := "test-image"
	dk.Spec.OneAgent = oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: imageID},
		},
	}
	dk.Status.CodeModules.ImageID = imageID

	return dk
}

func createDynaKubeWithJobFF(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	imageID := "test-image"
	dk.Spec.OneAgent = oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: imageID},
		},
	}
	dk.Status.CodeModules.ImageID = imageID
	dk.Annotations = map[string]string{
		exp.OANodeImagePullKey: "true",
	}

	return dk
}

func createNotReadyDynaKube(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	dk.Spec.OneAgent = oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
	}

	return dk
}

func createDynaKubeNoCSI(t *testing.T) *dynakube.DynaKube {
	t.Helper()

	dk := createDynaKubeBase(t)
	dk.Spec.OneAgent = oneagent.Spec{
		ClassicFullStack: &oneagent.HostInjectSpec{},
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
	m.EXPECT().InstallAgent(mock.Anything, mock.Anything).Return(true, nil)

	return m
}

func createNotReadyInstaller(t *testing.T) *installermock.Installer {
	t.Helper()

	m := installermock.NewInstaller(t)
	m.EXPECT().InstallAgent(mock.Anything, mock.Anything).Return(false, nil)

	return m
}

func createFailingInstaller(t *testing.T) *installermock.Installer {
	t.Helper()

	m := installermock.NewInstaller(t)
	m.EXPECT().InstallAgent(mock.Anything, mock.Anything).Return(false, errors.New("BOOM"))

	return m
}

func mockURLInstallerBuilder(t *testing.T, mockedInstaller *installermock.Installer) urlInstallerBuilder {
	t.Helper()

	return func(_ oneagentclient.APIClient, _ *url.Properties) installer.Installer {
		return mockedInstaller
	}
}

func mockImageInstallerBuilder(t *testing.T, mockedInstaller *installermock.Installer) imageInstallerBuilder {
	t.Helper()

	return func(_ context.Context, _ *image.Properties) (installer.Installer, error) {
		return mockedInstaller, nil
	}
}

func mockJobInstallerBuilder(t *testing.T, mockedInstaller *installermock.Installer, pullSecrets ...string) jobInstallerBuilder {
	t.Helper()

	return func(_ context.Context, props *job.Properties) installer.Installer {
		for _, pullSecret := range pullSecrets {
			assert.Contains(t, props.PullSecrets, pullSecret)
		}

		return mockedInstaller
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
			dtclient.APIToken: []byte("this is a token"),
		},
	}
}

func mockFailingDTClientBuilder(t *testing.T) dynatraceclient.BuilderV2 {
	t.Helper()

	mockDtcBuilder := dtbuildermock.NewBuilderV2(t)
	mockDtcBuilder.EXPECT().SetDynakube(mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().SetTokens(mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().SetUserAgentSuffix("provisioner").Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().Build(mock.Anything).Return(nil, errors.New("BOOM"))

	return mockDtcBuilder
}

func mockSuccessfulDTClientBuilder(t *testing.T) dynatraceclient.BuilderV2 {
	t.Helper()

	mockDtcBuilder := dtbuildermock.NewBuilderV2(t)
	mockDtcBuilder.EXPECT().SetDynakube(mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().SetTokens(mock.Anything).Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().SetUserAgentSuffix("provisioner").Return(mockDtcBuilder)
	mockDtcBuilder.EXPECT().Build(mock.Anything).Return(&dtclient.ClientV2{}, nil)

	return mockDtcBuilder
}
