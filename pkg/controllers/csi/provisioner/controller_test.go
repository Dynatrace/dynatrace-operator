package csiprovisioner

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	dtClientMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	reconcilermock "github.com/Dynatrace/dynatrace-operator/test/mocks/sigs.k8s.io/controller-runtime/pkg/reconcile"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
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
	ctx := context.Background()
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
		dynakube := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		_ = db.InsertDynakube(ctx, &dynakube)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        db,
			gc:        gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakube.Name}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)

		ten, err := db.GetDynakube(ctx, dynakube.TenantUUID)
		require.NoError(t, err)
		require.Nil(t, ten)
	})
	t.Run("application monitoring disabled", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{},
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
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
								AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{},
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
		_ = db.InsertDynakube(ctx, &metadata.Dynakube{Name: dynakubeName})
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
								AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{},
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

		dynakubeMetadatas, err := db.GetAllDynakubes(ctx)
		require.NoError(t, err)
		require.Empty(t, dynakubeMetadatas)
	})
	t.Run("host monitoring used", func(t *testing.T) {
		fakeClient := fake.NewClient(
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: dkName,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testAPIURL,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
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
		mockDtcBuilder := dtClientMock.NewBuilder(t)

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

		dynakubeMetadatas, err := db.GetAllDynakubes(ctx)
		require.NoError(t, err)
		require.Len(t, dynakubeMetadatas, 1)
	})
	t.Run("no tokens", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta1.DynaKubeStatus{
						CodeModules: dynatracev1beta1.CodeModulesStatus{
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
		mockDtcBuilder := dtClientMock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("Build").Return(nil, fmt.Errorf(errorMsg))

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta1.DynaKubeStatus{
						CodeModules: dynatracev1beta1.CodeModulesStatus{
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
		mockDtcBuilder := dtClientMock.NewBuilder(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta1.OneAgentSpec{
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
		mockDtcBuilder := dtClientMock.NewBuilder(t)
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
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
						connectioninfo.TenantTokenName: []byte("tenant-token"),
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
		mockDtcBuilder := dtClientMock.NewBuilder(t)

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta1.DynaKubeStatus{
						CodeModules: dynatracev1beta1.CodeModulesStatus{
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
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
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
		dk := &dynatracev1beta1.DynaKube{}

		isEnabled := dk.NeedsCSIDriver()

		require.False(t, isEnabled)
	})

	t.Run("application monitoring enabled", func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
		}

		isEnabled := dk.NeedsCSIDriver()

		require.True(t, isEnabled)
	})

	t.Run("application monitoring enabled without csi driver", func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{},
					},
				},
			},
		}

		isEnabled := dk.NeedsCSIDriver()

		require.False(t, isEnabled)
	})
}

func buildValidApplicationMonitoringSpec(_ *testing.T) *dynatracev1beta1.ApplicationMonitoringSpec {
	useCSIDriver := true

	return &dynatracev1beta1.ApplicationMonitoringSpec{
		UseCSIDriver: &useCSIDriver,
	}
}

func TestProvisioner_CreateDynakube(t *testing.T) {
	ctx := context.TODO()
	db := metadata.FakeMemoryDB()
	expectedOtherDynakube := metadata.NewDynakube(otherDkName, tenantUUID, "v1", "", 0)
	_ = db.InsertDynakube(ctx, expectedOtherDynakube)
	provisioner := &OneAgentProvisioner{
		db: db,
	}

	oldDynakube := metadata.Dynakube{}
	newDynakube := metadata.NewDynakube(dkName, tenantUUID, "v1", "", 0)

	err := provisioner.createOrUpdateDynakubeMetadata(ctx, oldDynakube, newDynakube)
	require.NoError(t, err)

	dynakube, err := db.GetDynakube(ctx, dkName)
	require.NoError(t, err)
	require.NotNil(t, dynakube)
	require.Equal(t, *newDynakube, *dynakube)

	otherDynakube, err := db.GetDynakube(ctx, otherDkName)
	require.NoError(t, err)
	require.NotNil(t, dynakube)
	require.Equal(t, *expectedOtherDynakube, *otherDynakube)
}

func TestProvisioner_UpdateDynakube(t *testing.T) {
	ctx := context.TODO()
	db := metadata.FakeMemoryDB()
	oldDynakube := metadata.NewDynakube(dkName, tenantUUID, "v1", "", 0)
	_ = db.InsertDynakube(ctx, oldDynakube)
	expectedOtherDynakube := metadata.NewDynakube(otherDkName, tenantUUID, "v1", "", 0)
	_ = db.InsertDynakube(ctx, expectedOtherDynakube)

	provisioner := &OneAgentProvisioner{
		db: db,
	}
	newDynakube := metadata.NewDynakube(dkName, "new-uuid", "v2", "", 0)

	err := provisioner.createOrUpdateDynakubeMetadata(ctx, *oldDynakube, newDynakube)
	require.NoError(t, err)

	dynakube, err := db.GetDynakube(ctx, dkName)
	require.NoError(t, err)
	require.NotNil(t, dynakube)
	require.Equal(t, *newDynakube, *dynakube)

	otherDynakube, err := db.GetDynakube(ctx, otherDkName)
	require.NoError(t, err)
	require.NotNil(t, dynakube)
	require.Equal(t, *expectedOtherDynakube, *otherDynakube)
}

func TestHandleMetadata(t *testing.T) {
	ctx := context.TODO()
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: dkName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testAPIURL,
		},
	}
	provisioner := &OneAgentProvisioner{
		db: metadata.FakeMemoryDB(),
	}
	dynakubeMetadata, oldMetadata, err := provisioner.handleMetadata(ctx, instance)

	require.NoError(t, err)
	require.NotNil(t, dynakubeMetadata)
	require.NotNil(t, oldMetadata)
	require.Equal(t, dynatracev1beta1.DefaultMaxFailedCsiMountAttempts, dynakubeMetadata.MaxFailedMountAttempts)

	instance.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureMaxFailedCsiMountAttempts: "5"}
	dynakubeMetadata, oldMetadata, err = provisioner.handleMetadata(ctx, instance)

	require.NoError(t, err)
	require.NotNil(t, dynakubeMetadata)
	require.NotNil(t, oldMetadata)
	require.Equal(t, 5, dynakubeMetadata.MaxFailedMountAttempts)
}
