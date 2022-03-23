package csiprovisioner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	dkName       = "dynakube-test"
	otherDkName  = "other-dk"
	errorMsg     = "test-error"
	tenantUUID   = "test-uid"
	agentVersion = "12345"
)

type mkDirAllErrorFs struct {
	afero.Fs
}

func (fs *mkDirAllErrorFs) MkdirAll(_ string, _ os.FileMode) error {
	return fmt.Errorf(errorMsg)
}

func TestOneAgentProvisioner_Reconcile(t *testing.T) {
	t.Run(`no dynakube instance`, func(t *testing.T) {
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        metadata.FakeMemoryDB(),
			mutex:     &sync.Mutex{},
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`dynakube deleted`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		dynakube := metadata.Dynakube{TenantUUID: tenantUUID, LatestVersion: agentVersion, Name: dkName}
		_ = db.InsertDynakube(&dynakube)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        db,
			mutex:     &sync.Mutex{},
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakube.Name}})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)

		ten, err := db.GetDynakube(dynakube.TenantUUID)
		assert.NoError(t, err)
		assert.Nil(t, ten)
	})
	t.Run(`application monitoring disabled`, func(t *testing.T) {
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{},
					},
				},
			),
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Minute}, result)
	})
	t.Run(`csi driver disabled`, func(t *testing.T) {
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
								AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{},
							},
						},
					},
				},
			),
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Minute}, result)
	})
	t.Run(`no tokens`, func(t *testing.T) {
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta1.DynaKubeStatus{
						ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
							TenantUUID: tenantUUID,
						},
					},
				},
			),
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, `failed to query tokens: secrets "`+dkName+`" not found`)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when creating dynatrace client`, func(t *testing.T) {
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
					Status: dynatracev1beta1.DynaKubeStatus{
						ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
							TenantUUID: tenantUUID,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
				return nil, fmt.Errorf(errorMsg)
			},
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to create Dynatrace client: "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when querying dynatrace client for connection info`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{}, fmt.Errorf(errorMsg))

		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(
				&dynatracev1beta1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: dynatracev1beta1.DynaKubeSpec{
						OneAgent: dynatracev1beta1.OneAgentSpec{
							ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
				return mockClient, nil
			},
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 15 * time.Second}, result)
	})
	t.Run(`error creating directories`, func(t *testing.T) {
		errorfs := &mkDirAllErrorFs{
			Fs: afero.NewMemMapFs(),
		}
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		testDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(&testDynakube,
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
				return mockClient, nil
			},
			db:    metadata.FakeMemoryDB(),
			mutex: &sync.Mutex{},
		}
		result, err := provisioner.reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}}, errorfs, testDynakube)

		assert.EqualError(t, err, "failed to create directory "+filepath.Join(tenantUUID)+": "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error getting latest agent version`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		mockClient.On("GetAgent",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("*mem.File")).Return(fmt.Errorf(errorMsg))
		mockClient.
			On("GetAgentVersions", dtclient.OsUnix, dtclient.InstallerTypePaaS, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(make([]string, 0), fmt.Errorf(errorMsg))
		mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("uint")).Return(&testProcessModuleConfig, nil)
		testDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(&testDynakube,
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
				return mockClient, nil
			},
			db:       metadata.FakeMemoryDB(),
			recorder: &record.FakeRecorder{},
			mutex:    &sync.Mutex{},
		}

		result, err := provisioner.reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}}, memFs, testDynakube)

		// "go test" breaks if the output does not end with a newline
		// making sure one is printed here
		log.Info("")

		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		exists, err := afero.Exists(memFs, tenantUUID)

		assert.NoError(t, err)
		assert.True(t, exists)
	})
	t.Run(`error getting dynakube from db`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(agentVersion, nil)
		testDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
			},
		}
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(&testDynakube,
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
				return mockClient, nil
			},
			db:    &metadata.FakeFailDB{},
			mutex: &sync.Mutex{},
		}

		result, err := provisioner.reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}}, memFs, testDynakube)

		assert.Error(t, err)
		assert.Empty(t, result)

	})
	t.Run(`correct directories are created`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		memDB := metadata.FakeMemoryDB()
		err := memDB.InsertDynakube(metadata.NewDynakube(dkName, tenantUUID, agentVersion))
		require.NoError(t, err)

		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(agentVersion, nil)
		mockClient.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, mock.AnythingOfType("string"),
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(5).(io.Writer)

				zipFile := setupTestZip(t, memFs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("uint")).Return(&testProcessModuleConfig, nil)
		testDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID: tenantUUID,
				},
				LatestAgentVersionUnixPaas: agentVersion,
			},
		}
		r := &OneAgentProvisioner{
			apiReader: fake.NewClient(&testDynakube,
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
				return mockClient, nil
			},
			db:       memDB,
			recorder: &record.FakeRecorder{},
			mutex:    &sync.Mutex{},
		}

		result, err := r.reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}}, memFs, testDynakube)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 5 * time.Minute}, result)

		exists, err := afero.Exists(memFs, tenantUUID)

		assert.NoError(t, err)
		assert.True(t, exists)

		fileInfo, err := memFs.Stat(tenantUUID)

		assert.NoError(t, err)
		assert.True(t, fileInfo.IsDir())
	})
}

func TestHasCodeModulesWithCSIVolumeEnabled(t *testing.T) {
	t.Run(`default DynaKube object returns false`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{}

		isEnabled := dk.NeedsCSIDriver()

		assert.False(t, isEnabled)
	})

	t.Run(`application monitoring enabled`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: buildValidApplicationMonitoringSpec(t),
				},
			},
		}

		isEnabled := dk.NeedsCSIDriver()

		assert.True(t, isEnabled)
	})

	t.Run(`application monitoring enabled without csi driver`, func(t *testing.T) {
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

		assert.False(t, isEnabled)
	})
}

func buildValidApplicationMonitoringSpec(_ *testing.T) *dynatracev1beta1.ApplicationMonitoringSpec {
	useCSIDriver := true
	return &dynatracev1beta1.ApplicationMonitoringSpec{
		UseCSIDriver: &useCSIDriver,
	}
}

func TestProvisioner_CreateDynakube(t *testing.T) {
	db := metadata.FakeMemoryDB()
	expectedOtherDynakube := metadata.NewDynakube(otherDkName, tenantUUID, "v1")
	db.InsertDynakube(expectedOtherDynakube)
	provisioner := &OneAgentProvisioner{
		db:    db,
		mutex: &sync.Mutex{},
	}

	oldDynakube := metadata.Dynakube{}
	newDynakube := metadata.NewDynakube(dkName, tenantUUID, "v1")

	err := provisioner.createOrUpdateDynakube(oldDynakube, newDynakube)
	require.NoError(t, err)

	dynakube, err := db.GetDynakube(dkName)
	assert.NoError(t, err)
	assert.NotNil(t, dynakube)
	assert.Equal(t, *newDynakube, *dynakube)

	otherDynakube, err := db.GetDynakube(otherDkName)
	assert.NoError(t, err)
	assert.NotNil(t, dynakube)
	assert.Equal(t, *expectedOtherDynakube, *otherDynakube)
}

func TestProvisioner_UpdateDynakube(t *testing.T) {
	db := metadata.FakeMemoryDB()
	oldDynakube := metadata.NewDynakube(dkName, tenantUUID, "v1")
	db.InsertDynakube(oldDynakube)
	expectedOtherDynakube := metadata.NewDynakube(otherDkName, tenantUUID, "v1")
	db.InsertDynakube(expectedOtherDynakube)

	provisioner := &OneAgentProvisioner{
		db:    db,
		mutex: &sync.Mutex{},
	}
	newDynakube := metadata.NewDynakube(dkName, "new-uuid", "v2")

	err := provisioner.createOrUpdateDynakube(*oldDynakube, newDynakube)
	require.NoError(t, err)

	dynakube, err := db.GetDynakube(dkName)
	assert.NoError(t, err)
	assert.NotNil(t, dynakube)
	assert.Equal(t, *newDynakube, *dynakube)

	otherDynakube, err := db.GetDynakube(otherDkName)
	assert.NoError(t, err)
	assert.NotNil(t, dynakube)
	assert.Equal(t, *expectedOtherDynakube, *otherDynakube)
}
