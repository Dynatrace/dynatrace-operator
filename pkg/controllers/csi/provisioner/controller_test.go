package csiprovisioner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	reconcilermock "github.com/Dynatrace/dynatrace-operator/test/mocks/sigs.k8s.io/controller-runtime/pkg/reconcile"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testAPIURL    = "http://test-uid/api"
	tenantUUID    = "test-uid"
	dkName        = "dynakube-test"
	testNamespace = "test-namespace"
	testVersion   = "1.2.3"
	testImageID   = "test-image-id"
	otherDkName   = "other-dk"
	errorMsg      = "test-error"
	agentVersion  = "12345"
	testRuxitConf = `
[general]
key value
`
)

type mkDirAllErrorFs struct {
	afero.Fs
}

func (fs *mkDirAllErrorFs) MkdirAll(_ string, _ os.FileMode) error {
	return fmt.Errorf(errorMsg)
}

func TestOneAgentProvisioner_Reconcile(t *testing.T) {
	dynakubeName := "test-dk"

	t.Run("no dynakube instance", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        metadata.FakeMemoryDB(),
			gc:        gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)
	})
	t.Run("dynakube deleted", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		db := metadata.FakeMemoryDB()

		tenantConfig := metadata.TenantConfig{
			TenantUUID:                  tenantUUID,
			Name:                        dkName,
			DownloadedCodeModuleVersion: agentVersion,
		}

		err := db.CreateTenantConfig(&tenantConfig)
		require.NoError(t, err)

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        db,
			gc:        gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: tenantConfig.Name}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)

		ten, err := db.ReadTenantConfig(metadata.TenantConfig{TenantUUID: tenantConfig.TenantUUID})
		require.Error(t, err)
		require.Nil(t, ten)
	})
	t.Run("application monitoring disabled", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						OneAgent: dynatracev1beta2.OneAgentSpec{},
					},
				},
			),
			db: metadata.FakeMemoryDB(),
			gc: gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakubeName}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)
	})
	t.Run("csi driver not enabled", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
								AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{},
							},
						},
					},
				},
			),
			db: metadata.FakeMemoryDB(),
			gc: gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakubeName}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)
	})
	t.Run("csi driver disabled", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		db := metadata.FakeMemoryDB()
		_ = db.CreateTenantConfig(&metadata.TenantConfig{Name: dynakubeName})
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
								AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{},
							},
						},
					},
				},
			),
			db: db,
			gc: gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakubeName}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)

		tenantConfigs, err := db.ReadTenantConfigs()
		require.NoError(t, err)
		require.Empty(t, tenantConfigs)
	})
	t.Run("host monitoring used", func(t *testing.T) {
		fakeClient := fake.NewClient(
			&dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: dkName,
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testAPIURL,
					OneAgent: dynatracev1beta2.OneAgentSpec{
						HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: dkName,
				},
				Data: map[string][]byte{
					dtclient.ApiToken: []byte("api-token"),
				},
			},
		)
		mockDtcBuilder := dtbuildermock.NewBuilder(t)

		gc := reconcilermock.NewReconciler(t)
		db := metadata.FakeMemoryDB()

		provisioner := &OneAgentProvisioner{
			apiReader:              fakeClient,
			client:                 fakeClient,
			fs:                     afero.NewMemMapFs(),
			db:                     db,
			gc:                     gc,
			path:                   metadata.PathResolver{},
			dynatraceClientBuilder: mockDtcBuilder,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)

		tenantConfigs, err := db.ReadTenantConfigs()
		require.NoError(t, err)
		require.Len(t, tenantConfigs, 1)
	})
	t.Run("no tokens", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta2.DynaKubeStatus{
						CodeModules: dynatracev1beta2.CodeModulesStatus{
							VersionStatus: status.VersionStatus{
								Version: "1.2.3",
							},
						},
					},
				},
			),
			gc: gc,
			db: metadata.FakeMemoryDB(),
			fs: afero.NewMemMapFs(),
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.EqualError(t, err, `secrets "`+dkName+`" not found`)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)
	})
	t.Run("error when creating dynatrace client", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("Build").Return(nil, fmt.Errorf(errorMsg))

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta2.DynaKubeStatus{
						CodeModules: dynatracev1beta2.CodeModulesStatus{
							VersionStatus: status.VersionStatus{
								Version: "1.2.3",
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.ApiToken: []byte("test-value"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			gc:                     gc,
			db:                     metadata.FakeMemoryDB(),
			fs:                     afero.NewMemMapFs(),
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.EqualError(t, err, "failed to create Dynatrace client: "+errorMsg)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)
	})
	t.Run("error creating directories", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		errorfs := &mkDirAllErrorFs{
			Fs: afero.NewMemMapFs(),
		}
		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.ApiToken: []byte("api-token"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			fs:                     errorfs,
			db:                     metadata.FakeMemoryDB(),
			gc:                     gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.EqualError(t, err, "failed to create directory "+tenantUUID+": "+errorMsg)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)

		// Logging newline so go test can parse the output correctly
		log.Info("")
	})
	t.Run("error getting latest agent version", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		memFs := afero.NewMemMapFs()
		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		dynakube := &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
		}
		installerMock := installermock.NewInstaller(t)

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				dynakube,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.ApiToken: []byte("api-token"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakube.OneagentTenantSecret(),
					},
					Data: map[string][]byte{
						connectioninfo.TenantTokenKey: []byte("tenant-token"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			fs:                     memFs,
			db:                     metadata.FakeMemoryDB(),
			recorder:               &record.FakeRecorder{},
			gc:                     gc,
			urlInstallerBuilder:    mockUrlInstallerBuilder(installerMock),
		}

		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		// "go test" breaks if the output does not end with a newline
		// making sure one is printed here
		log.Info("")

		require.NoError(t, err)
		require.NotEmpty(t, result)

		exists, err := afero.Exists(memFs, tenantUUID)

		require.NoError(t, err)
		require.True(t, exists)
	})
	t.Run("error getting dynakube from db", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		memFs := afero.NewMemMapFs()
		mockDtcBuilder := dtbuildermock.NewBuilder(t)

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta2.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta2.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta2.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta2.DynaKubeStatus{
						CodeModules: dynatracev1beta2.CodeModulesStatus{
							VersionStatus: status.VersionStatus{
								Version: "1.2.3",
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			fs:                     memFs,
			db:                     &metadata.FakeFailDB{},
			gc:                     gc,
		}

		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.Error(t, err)
		require.Empty(t, result)
	})
	t.Run("correct directories are created", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		memFs := afero.NewMemMapFs()
		memDB := metadata.FakeMemoryDB()
		dynakube := &dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}

		r := &OneAgentProvisioner{
			apiReader: fake.NewClient(dynakube),
			fs:        memFs,
			db:        memDB,
			gc:        gc,
		}

		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.NoError(t, err)
		require.NotNil(t, result)

		exists, err := afero.Exists(memFs, tenantUUID)

		require.NoError(t, err)
		require.True(t, exists)

		fileInfo, err := memFs.Stat(tenantUUID)

		require.NoError(t, err)
		require.True(t, fileInfo.IsDir())
	})
}

func TestHasCodeModulesWithCSIVolumeEnabled(t *testing.T) {
	t.Run("default DynaKube object returns false", func(t *testing.T) {
		dk := &dynatracev1beta2.DynaKube{}

		isEnabled := dk.NeedsCSIDriver()

		require.False(t, isEnabled)
	})

	t.Run("application monitoring enabled", func(t *testing.T) {
		dk := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
		}

		isEnabled := dk.NeedsCSIDriver()

		require.True(t, isEnabled)
	})

	t.Run("application monitoring enabled without csi driver", func(t *testing.T) {
		dk := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{
						AppInjectionSpec: dynatracev1beta2.AppInjectionSpec{},
					},
				},
			},
		}

		isEnabled := dk.NeedsCSIDriver()

		require.False(t, isEnabled)
	})
}

func buildValidApplicationMonitoringSpec(_ *testing.T) *dynatracev1beta2.ApplicationMonitoringSpec {
	useCSIDriver := true

	return &dynatracev1beta2.ApplicationMonitoringSpec{
		UseCSIDriver: &useCSIDriver,
	}
}

func TestProvisioner_CreateTenantConfig(t *testing.T) {
	db := metadata.FakeMemoryDB()

	expectedOtherTenantConfig := metadata.TenantConfig{Name: otherDkName, TenantUUID: tenantUUID, DownloadedCodeModuleVersion: "v1", MaxFailedMountAttempts: 0}
	db.CreateTenantConfig(&expectedOtherTenantConfig)

	provisioner := &OneAgentProvisioner{
		db: db,
	}

	newTenantConfig := metadata.TenantConfig{Name: dkName, TenantUUID: tenantUUID, DownloadedCodeModuleVersion: "v1", MaxFailedMountAttempts: 0}

	err := provisioner.db.UpdateTenantConfig(&newTenantConfig)
	require.NoError(t, err)

	storedTenantConfig, err := db.ReadTenantConfig(metadata.TenantConfig{Name: dkName})
	require.NoError(t, err)
	require.NotNil(t, storedTenantConfig)

	newTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	storedTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	require.Equal(t, newTenantConfig, *storedTenantConfig)

	storedTenantConfig, err = db.ReadTenantConfig(metadata.TenantConfig{Name: otherDkName})
	require.NoError(t, err)
	require.NotNil(t, storedTenantConfig)

	expectedOtherTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	storedTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	require.Equal(t, expectedOtherTenantConfig, *storedTenantConfig)
}

func TestProvisioner_UpdateDynakube(t *testing.T) {
	db := metadata.FakeMemoryDB()

	oldTenantConfig := metadata.TenantConfig{Name: dkName, TenantUUID: tenantUUID, DownloadedCodeModuleVersion: "v1", MaxFailedMountAttempts: 0}
	_ = db.CreateTenantConfig(&oldTenantConfig)
	expectedOtherTenantConfig := metadata.TenantConfig{Name: otherDkName, TenantUUID: tenantUUID, DownloadedCodeModuleVersion: "v1", MaxFailedMountAttempts: 0}
	_ = db.CreateTenantConfig(&expectedOtherTenantConfig)

	provisioner := &OneAgentProvisioner{
		db: db,
	}
	newTenantConfig := metadata.TenantConfig{UID: oldTenantConfig.UID, Name: dkName, TenantUUID: "new-uuid", DownloadedCodeModuleVersion: "v2", MaxFailedMountAttempts: 0}

	err := provisioner.db.UpdateTenantConfig(&newTenantConfig)
	require.NoError(t, err)

	tenantConfig, err := db.ReadTenantConfig(metadata.TenantConfig{Name: dkName})
	require.NoError(t, err)
	require.NotNil(t, tenantConfig)

	newTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	tenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	require.Equal(t, newTenantConfig, *tenantConfig)

	otherTenantConfig, err := db.ReadTenantConfig(metadata.TenantConfig{Name: otherDkName})
	require.NoError(t, err)
	require.NotNil(t, otherTenantConfig)

	expectedOtherTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	otherTenantConfig.TimeStampedModel = metadata.TimeStampedModel{}
	require.Equal(t, expectedOtherTenantConfig, *otherTenantConfig)
}

func TestHandleMetadata(t *testing.T) {
	dynakube := &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: dkName,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			APIURL: testAPIURL,
		},
	}
	provisioner := &OneAgentProvisioner{
		db: metadata.FakeMemoryDB(),
	}
	dynakubeMetadata, err := provisioner.handleMetadata(dynakube)

	require.NoError(t, err)
	require.NotNil(t, dynakubeMetadata)
	require.Equal(t, int64(dynatracev1beta2.DefaultMaxFailedCsiMountAttempts), dynakubeMetadata.MaxFailedMountAttempts)

	dynakube.Annotations = map[string]string{dynatracev1beta2.AnnotationFeatureMaxFailedCsiMountAttempts: "5"}
	dynakubeMetadata, err = provisioner.handleMetadata(dynakube)

	require.NoError(t, err)
	require.NotNil(t, dynakubeMetadata)
	require.Equal(t, int64(5), dynakubeMetadata.MaxFailedMountAttempts)
}

func TestUpdateAgentInstallation(t *testing.T) {
	ctx := context.Background()

	t.Run("updateAgentInstallation with codeModules enabled", func(t *testing.T) {
		dynakube := getDynakube()
		enableCodeModules(dynakube)

		mockDtcBuilder := dtbuildermock.NewBuilder(t)

		var dtc dtclient.Client

		mockDtcBuilder.On("Build").Return(dtc, nil)
		dtc, err := mockDtcBuilder.Build()
		require.NoError(t, err)

		mockK8sClient := createMockK8sClient(ctx, dynakube)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("*context.valueCtx"), "test/codemodules/test").
			Return(true, nil)

		provisioner := &OneAgentProvisioner{
			db:                     metadata.FakeMemoryDB(),
			dynatraceClientBuilder: mockDtcBuilder,
			apiReader:              mockK8sClient,
			client:                 mockK8sClient,
			path:                   metadata.PathResolver{RootDir: "test"},
			fs:                     afero.NewMemMapFs(),
			imageInstallerBuilder:  mockImageInstallerBuilder(installerMock),
			recorder:               &record.FakeRecorder{},
		}

		ruxitAgentProcPath := filepath.Join("test", "codemodules", "test", "agent", "conf", "ruxitagentproc.conf")
		sourceRuxitAgentProcPath := filepath.Join("test", "codemodules", "test", "agent", "conf", "_ruxitagentproc.conf")

		setUpFS(provisioner.fs, ruxitAgentProcPath, sourceRuxitAgentProcPath)

		mockRegistryClient(t, provisioner, "test")

		tenantConfig := metadata.TenantConfig{Name: dkName, TenantUUID: tenantUUID, DownloadedCodeModuleVersion: agentVersion}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &tenantConfig, dynakube)
		require.NoError(t, err)

		require.Equal(t, "test", tenantConfig.DownloadedCodeModuleVersion)
		assert.False(t, isRequeue)
	})
	t.Run("updateAgentInstallation with codeModules enabled errors and requeues", func(t *testing.T) {
		dynakube := getDynakube()
		enableCodeModules(dynakube)

		mockDtcBuilder := dtbuildermock.NewBuilder(t)

		var dtc dtclient.Client

		mockDtcBuilder.On("Build").Return(dtc, nil)
		dtc, err := mockDtcBuilder.Build()
		require.NoError(t, err)

		mockK8sClient := createMockK8sClient(ctx, dynakube)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("*context.valueCtx"), "test/codemodules/test").
			Return(true, nil)

		provisioner := &OneAgentProvisioner{
			db:                     metadata.FakeMemoryDB(),
			dynatraceClientBuilder: mockDtcBuilder,
			apiReader:              mockK8sClient,
			client:                 mockK8sClient,
			path:                   metadata.PathResolver{RootDir: "test"},
			fs:                     afero.NewMemMapFs(),
			imageInstallerBuilder:  mockImageInstallerBuilder(installerMock),
			recorder:               &record.FakeRecorder{},
		}

		mockRegistryClient(t, provisioner, "test")

		tenantConfig := metadata.TenantConfig{TenantUUID: tenantUUID, DownloadedCodeModuleVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &tenantConfig, dynakube)
		require.NoError(t, err)

		require.Equal(t, "12345", tenantConfig.DownloadedCodeModuleVersion)
		assert.True(t, isRequeue)
	})
	t.Run("updateAgentInstallation without codeModules", func(t *testing.T) {
		dynakube := getDynakube()

		mockDtcBuilder := dtbuildermock.NewBuilder(t)

		var dtc dtclient.Client

		mockDtcBuilder.On("Build").Return(dtc, nil)
		dtc, err := mockDtcBuilder.Build()
		require.NoError(t, err)

		mockK8sClient := createMockK8sClient(ctx, dynakube)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("*context.valueCtx"), "test/codemodules").
			Return(true, nil)

		provisioner := &OneAgentProvisioner{
			db:                     metadata.FakeMemoryDB(),
			dynatraceClientBuilder: mockDtcBuilder,
			apiReader:              mockK8sClient,
			client:                 mockK8sClient,
			path:                   metadata.PathResolver{RootDir: "test"},
			fs:                     afero.NewMemMapFs(),
			recorder:               &record.FakeRecorder{},
			urlInstallerBuilder:    mockUrlInstallerBuilder(installerMock),
		}
		ruxitAgentProcPath := filepath.Join("test", "codemodules", "agent", "conf", "ruxitagentproc.conf")
		sourceRuxitAgentProcPath := filepath.Join("test", "codemodules", "agent", "conf", "_ruxitagentproc.conf")

		setUpFS(provisioner.fs, ruxitAgentProcPath, sourceRuxitAgentProcPath)

		mockRegistryClient(t, provisioner, "test")

		tenantConfig := metadata.TenantConfig{TenantUUID: tenantUUID, DownloadedCodeModuleVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &tenantConfig, dynakube)
		require.NoError(t, err)

		require.Equal(t, "12345", tenantConfig.DownloadedCodeModuleVersion)
		assert.False(t, isRequeue)
	})
	t.Run("updateAgentInstallation without codeModules errors and requeues", func(t *testing.T) {
		dynakube := getDynakube()

		mockDtcBuilder := dtbuildermock.NewBuilder(t)

		var dtc dtclient.Client

		mockDtcBuilder.On("Build").Return(dtc, nil)
		dtc, err := mockDtcBuilder.Build()
		require.NoError(t, err)

		mockK8sClient := createMockK8sClient(ctx, dynakube)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("*context.valueCtx"), "test/codemodules").
			Return(true, nil)

		provisioner := &OneAgentProvisioner{
			db:                     metadata.FakeMemoryDB(),
			dynatraceClientBuilder: mockDtcBuilder,
			apiReader:              mockK8sClient,
			client:                 mockK8sClient,
			path:                   metadata.PathResolver{RootDir: "test"},
			fs:                     afero.NewMemMapFs(),
			recorder:               &record.FakeRecorder{},
			urlInstallerBuilder:    mockUrlInstallerBuilder(installerMock),
		}

		mockRegistryClient(t, provisioner, "test")

		tenantConfig := metadata.TenantConfig{TenantUUID: tenantUUID, DownloadedCodeModuleVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &tenantConfig, dynakube)
		require.NoError(t, err)

		require.Equal(t, "12345", tenantConfig.DownloadedCodeModuleVersion)
		assert.True(t, isRequeue)
	})
}

func createMockK8sClient(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) client.Client {
	mockK8sClient := fake.NewClient(dynakube)
	mockK8sClient.Create(ctx,
		&corev1.Secret{
			Data: map[string][]byte{processmoduleconfigsecret.SecretKeyProcessModuleConfig: []byte(`{"revision":0,"properties":[]}`)},
			ObjectMeta: metav1.ObjectMeta{
				Name:      strings.Join([]string{dkName, "pmc-secret"}, "-"),
				Namespace: "test-namespace",
			},
		},
	)

	return mockK8sClient
}

func getDynakube() *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			APIURL:   testAPIURL,
			OneAgent: dynatracev1beta2.OneAgentSpec{},
		},
	}
}

func enableCodeModules(dynakube *dynatracev1beta2.DynaKube) {
	dynakube.Status = dynatracev1beta2.DynaKubeStatus{
		CodeModules: dynatracev1beta2.CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				Version: testVersion,
				ImageID: testImageID,
			},
		},
	}
}

func setUpFS(fs afero.Fs, ruxitAgentProcPath string, sourceRuxitAgentProcPath string) {
	_ = fs.MkdirAll(filepath.Base(sourceRuxitAgentProcPath), 0755)
	_ = fs.MkdirAll(filepath.Base(ruxitAgentProcPath), 0755)
	usedConf, _ := fs.OpenFile(ruxitAgentProcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	_, _ = usedConf.WriteString(testRuxitConf)
}
