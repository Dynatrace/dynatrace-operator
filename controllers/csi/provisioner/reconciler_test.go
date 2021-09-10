package csiprovisioner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	dkName            = "dynakube-test"
	errorMsg          = "test-error"
	tenantUUID        = "test-uid"
	agentVersion      = "12345"
	invalidDriverName = "csi.not.dynatrace.com"
)

type mkDirAllErrorFs struct {
	afero.Fs
}

func (fs *mkDirAllErrorFs) MkdirAll(_ string, _ os.FileMode) error {
	return fmt.Errorf(errorMsg)
}

func TestOneAgentProvisioner_Reconcile(t *testing.T) {
	t.Run(`no dynakube instance`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(),
			db:     metadata.FakeMemoryDB(),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`dynakube deleted`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		tenant := metadata.Tenant{TenantUUID: tenantUUID, LatestVersion: agentVersion, Dynakube: dkName}
		_ = db.InsertTenant(&tenant)
		r := &OneAgentProvisioner{
			client: fake.NewClient(),
			db:     db,
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: tenant.Dynakube}})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)

		ten, err := db.GetTenant(tenant.TenantUUID)
		assert.NoError(t, err)
		assert.Nil(t, ten)
	})
	t.Run(`code modules disabled`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: false,
						},
					},
				},
			),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Minute}, result)
	})
	t.Run(`csi driver disabled`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
							Volume:  v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
						},
					},
				},
			),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Minute}, result)
	})
	t.Run(`no tokens`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
					Status: v1alpha1.DynaKubeStatus{
						ConnectionInfo: v1alpha1.ConnectionInfoStatus{
							TenantUUID: tenantUUID,
						},
					},
				},
			),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, `failed to query tokens: secrets "`+dkName+`" not found`)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when creating dynatrace client`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
					Status: v1alpha1.DynaKubeStatus{
						ConnectionInfo: v1alpha1.ConnectionInfoStatus{
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
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return nil, fmt.Errorf(errorMsg)
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to create Dynatrace client: "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when querying dynatrace client for connection info`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{}, fmt.Errorf(errorMsg))

		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

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
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
					Status: v1alpha1.DynaKubeStatus{
						ConnectionInfo: v1alpha1.ConnectionInfoStatus{
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
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			fs: errorfs,
			db: metadata.FakeMemoryDB(),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

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
			On("GetAgentVersions", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro, mock.AnythingOfType("string")).
			Return(make([]string, 0), fmt.Errorf(errorMsg))
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
					Status: v1alpha1.DynaKubeStatus{
						ConnectionInfo: v1alpha1.ConnectionInfoStatus{
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
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			fs:       memFs,
			db:       metadata.FakeMemoryDB(),
			recorder: &record.FakeRecorder{},
		}

		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		// "go test" breaks if the output does not end with a newline
		// making sure one is printed here
		log.Info("")

		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		exists, err := afero.Exists(memFs, tenantUUID)

		assert.NoError(t, err)
		assert.True(t, exists)
	})
	t.Run(`error getting tenant`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(agentVersion, nil)
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
					Status: v1alpha1.DynaKubeStatus{
						ConnectionInfo: v1alpha1.ConnectionInfoStatus{
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
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			fs: memFs,
			db: &metadata.FakeFailDB{},
		}

		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.Error(t, err)
		assert.Empty(t, result)

	})
	t.Run(`correct directories are created`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		memDB := metadata.FakeMemoryDB()
		err := memDB.InsertTenant(metadata.NewTenant(tenantUUID, agentVersion, dkName))
		require.NoError(t, err)

		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(agentVersion, nil)
		mockClient.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(5).(io.Writer)

				zipFile := setupTestZip(t, memFs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: buildValidCodeModulesSpec(t),
					},
					Status: v1alpha1.DynaKubeStatus{
						ConnectionInfo: v1alpha1.ConnectionInfoStatus{
							TenantUUID: tenantUUID,
						},
						LatestAgentVersionUnixPaas: agentVersion,
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			fs:       memFs,
			db:       memDB,
			recorder: &record.FakeRecorder{},
		}

		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

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
		dk := &v1alpha1.DynaKube{}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.False(t, isEnabled)
	})

	t.Run(`code modules disabled returns false`, func(t *testing.T) {
		dk := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: false,
				},
			},
		}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.False(t, isEnabled)
	})

	t.Run(`code modules enabled with no volume returns true`, func(t *testing.T) {
		dk := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.True(t, isEnabled)
	})

	t.Run(`code modules enabled with empty volume returns true`, func(t *testing.T) {
		dk := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: true,
					Volume:  v1.VolumeSource{},
				},
			},
		}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.True(t, isEnabled)
	})

	t.Run(`code modules enabled with csi.oneagent.dynatrace.com volume returns true`, func(t *testing.T) {
		dk := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: true,
					Volume:  v1.VolumeSource{CSI: &v1.CSIVolumeSource{Driver: dtcsi.DriverName}},
				},
			},
		}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.True(t, isEnabled)
	})

	t.Run(`code modules enabled with other csi volume returns false`, func(t *testing.T) {
		dk := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: true,
					Volume:  v1.VolumeSource{CSI: &v1.CSIVolumeSource{Driver: invalidDriverName}},
				},
			},
		}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.False(t, isEnabled)
	})

	t.Run(`code modules enabled with emptydir volume returns false`, func(t *testing.T) {
		dk := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: true,
					Volume:  v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
				},
			},
		}

		isEnabled := hasCodeModulesWithCSIVolumeEnabled(dk)

		assert.False(t, isEnabled)
	})
}

func buildValidCodeModulesSpec(_ *testing.T) v1alpha1.CodeModulesSpec {
	return v1alpha1.CodeModulesSpec{
		Enabled: true,
		Volume: v1.VolumeSource{
			CSI: &v1.CSIVolumeSource{
				Driver: dtcsi.DriverName,
			},
		},
	}
}
