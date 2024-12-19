package csiprovisioner

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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
	return errors.New(errorMsg)
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
		metadataDk := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		_ = db.InsertDynakube(ctx, &metadataDk)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        db,
			gc:        gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: metadataDk.Name}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)

		ten, err := db.GetDynakube(ctx, metadataDk.TenantUUID)
		require.NoError(t, err)
		require.Nil(t, ten)
	})
	t.Run("csi driver not used (classicFullstack)", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dynakubeName,
					},
					Spec: dynakube.DynaKubeSpec{
						OneAgent: dynakube.OneAgentSpec{
							ClassicFullStack: &dynakube.HostInjectSpec{},
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
	t.Run("host monitoring used", func(t *testing.T) {
		fakeClient := fake.NewClient(
			addFakeTenantUUID(
				&dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynakube.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynakube.OneAgentSpec{
							HostMonitoring: &dynakube.HostInjectSpec{},
						},
					},
				},
			),
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

		dynakubeMetadatas, err := db.GetAllDynakubes(ctx)
		require.NoError(t, err)
		require.Len(t, dynakubeMetadatas, 1)
	})
	t.Run("no tokens", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				addFakeTenantUUID(
					&dynakube.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name: dkName,
						},
						Spec: dynakube.DynaKubeSpec{
							APIURL: testAPIURL,
							OneAgent: dynakube.OneAgentSpec{
								ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
							},
						},
						Status: dynakube.DynaKubeStatus{
							CodeModules: dynakube.CodeModulesStatus{
								VersionStatus: status.VersionStatus{
									Version: "1.2.3",
								},
							},
						},
					},
				),
			),
			gc: gc,
			db: metadata.FakeMemoryDB(),
			fs: afero.NewMemMapFs(),
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.EqualError(t, err, `secrets "`+dkName+`" not found`)
		require.NotNil(t, result)
	})
	t.Run("error when creating dynatrace client", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("Build").Return(nil, errors.New(errorMsg))

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				addFakeTenantUUID(
					&dynakube.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name: dkName,
						},
						Spec: dynakube.DynaKubeSpec{
							APIURL: testAPIURL,
							OneAgent: dynakube.OneAgentSpec{
								ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
							},
						},
						Status: dynakube.DynaKubeStatus{
							CodeModules: dynakube.CodeModulesStatus{
								VersionStatus: status.VersionStatus{
									Version: "1.2.3",
								},
							},
						},
					},
				),
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
	})
	t.Run("error creating directories", func(t *testing.T) {
		gc := reconcilermock.NewReconciler(t)
		errorfs := &mkDirAllErrorFs{
			Fs: afero.NewMemMapFs(),
		}
		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				addFakeTenantUUID(
					&dynakube.DynaKube{
						ObjectMeta: metav1.ObjectMeta{
							Name: dkName,
						},
						Spec: dynakube.DynaKubeSpec{
							APIURL: testAPIURL,
							OneAgent: dynakube.OneAgentSpec{
								ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
							},
						},
					},
				),
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
		dynakube := addFakeTenantUUID(
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: dkName,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
					},
				},
			},
		)
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
				&dynakube.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynakube.DynaKubeSpec{
						APIURL: testAPIURL,
						OneAgent: dynakube.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynakube.DynaKubeStatus{
						CodeModules: dynakube.CodeModulesStatus{
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
		dynakube := addFakeTenantUUID(
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: dkName,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{},
					},
				},
			},
		)

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

func buildValidApplicationMonitoringSpec(_ *testing.T) *dynakube.ApplicationMonitoringSpec {
	return &dynakube.ApplicationMonitoringSpec{}
}

func TestProvisioner_CreateDynakube(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
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

		path := metadata.PathResolver{RootDir: "test"}
		base64Image := base64.StdEncoding.EncodeToString([]byte(dynakube.CodeModulesImage()))
		targetDir := path.AgentSharedBinaryDirForAgent(base64Image)

		mockK8sClient := createMockK8sClient(ctx, dynakube)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
			Return(true, nil)

		provisioner := &OneAgentProvisioner{
			db:                     metadata.FakeMemoryDB(),
			dynatraceClientBuilder: mockDtcBuilder,
			apiReader:              mockK8sClient,
			client:                 mockK8sClient,
			path:                   path,
			fs:                     afero.NewMemMapFs(),
			imageInstallerBuilder:  mockImageInstallerBuilder(installerMock),
			recorder:               &record.FakeRecorder{},
		}

		ruxitAgentProcPath := filepath.Join(targetDir, "agent", "conf", "ruxitagentproc.conf")
		sourceRuxitAgentProcPath := filepath.Join(targetDir, "agent", "conf", "_ruxitagentproc.conf")

		setUpFS(provisioner.fs, ruxitAgentProcPath, sourceRuxitAgentProcPath)

		dynakubeMetadata := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &dynakubeMetadata, dynakube)
		require.NoError(t, err)

		require.Equal(t, "", dynakubeMetadata.LatestVersion)
		require.Equal(t, base64Image, dynakubeMetadata.ImageDigest)
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

		path := metadata.PathResolver{RootDir: "test"}
		base64Image := base64.StdEncoding.EncodeToString([]byte(dynakube.CodeModulesImage()))
		targetDir := path.AgentSharedBinaryDirForAgent(base64Image)

		mockK8sClient := createMockK8sClient(ctx, dynakube)
		installerMock := installermock.NewInstaller(t)
		installerMock.
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).
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

		dynakubeMetadata := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &dynakubeMetadata, dynakube)
		require.NoError(t, err)

		require.Equal(t, "12345", dynakubeMetadata.LatestVersion)
		require.Equal(t, "", dynakubeMetadata.ImageDigest)
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
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), "test/codemodules").
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

		dynakubeMetadata := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &dynakubeMetadata, dynakube)
		require.NoError(t, err)

		require.Equal(t, "12345", dynakubeMetadata.LatestVersion)
		require.Equal(t, "", dynakubeMetadata.ImageDigest)
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
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), "test/codemodules").
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

		dynakubeMetadata := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		isRequeue, err := provisioner.updateAgentInstallation(ctx, dtc, &dynakubeMetadata, dynakube)
		require.NoError(t, err)

		require.Equal(t, "12345", dynakubeMetadata.LatestVersion)
		require.Equal(t, "", dynakubeMetadata.ImageDigest)
		assert.True(t, isRequeue)
	})
}

func createMockK8sClient(ctx context.Context, dynakube *dynakube.DynaKube) client.Client {
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

func getDynakube() *dynakube.DynaKube {
	return addFakeTenantUUID(&dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:   testAPIURL,
			OneAgent: dynakube.OneAgentSpec{},
		},
	})
}

func enableCodeModules(dk *dynakube.DynaKube) {
	dk.Status.CodeModules = dynakube.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			Version: testVersion,
			ImageID: testImageID,
		},
	}
}

func addFakeTenantUUID(dynakube *dynakube.DynaKube) *dynakube.DynaKube {
	dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = tenantUUID

	return dynakube
}

func setUpFS(fs afero.Fs, ruxitAgentProcPath string, sourceRuxitAgentProcPath string) {
	_ = fs.MkdirAll(filepath.Base(sourceRuxitAgentProcPath), 0755)
	_ = fs.MkdirAll(filepath.Base(ruxitAgentProcPath), 0755)
	usedConf, _ := fs.OpenFile(ruxitAgentProcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	_, _ = usedConf.WriteString(testRuxitConf)
}
