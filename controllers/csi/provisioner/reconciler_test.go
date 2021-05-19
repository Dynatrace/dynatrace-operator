package csiprovisioner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	dkName          = "dynakube-test"
	errorMsg        = "test-error"
	tenantUUID      = "test-uid"
	agentVersion    = "12345"
	dkTenantMapping = "tenant-dynakube-test"
)

type mkDirAllErrorFs struct {
	afero.Fs
}

func (fs *mkDirAllErrorFs) MkdirAll(_ string, _ os.FileMode) error {
	return fmt.Errorf(errorMsg)
}

type readFileErrorFs struct {
	afero.Fs
}

func (fs *readFileErrorFs) Open(_ string) (afero.File, error) {
	return nil, fmt.Errorf(errorMsg)
}

type readVersionFileErrorFs struct {
	afero.Fs
}

func (fs *readVersionFileErrorFs) Open(name string) (afero.File, error) {
	if strings.HasSuffix(name, dtcsi.VersionDir) {
		return nil, fmt.Errorf(errorMsg)
	}
	return fs.Fs.Open(name)
}

func TestOneAgentProvisioner_Reconcile(t *testing.T) {
	t.Run(`no dynakube instance`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`code modules not enabled`, func(t *testing.T) {
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
	t.Run(`no tokens`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to fetch connection info: "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
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
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to create directory "+filepath.Join(dtcsi.DataPath, tenantUUID)+": "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error reading tenant file`, func(t *testing.T) {
		errorFs := &readFileErrorFs{
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
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
			fs: errorFs,
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to query assigned DynaKube tenant: "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error getting latest agent version`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: tenantUUID,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return("", fmt.Errorf(errorMsg))
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to query OneAgent version: "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)

		tenantPath := filepath.Join(dtcsi.DataPath, tenantUUID)
		exists, err := afero.Exists(memFs, tenantPath)

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(memFs, filepath.Join(tenantPath, dtcsi.LogDir))

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(memFs, filepath.Join(tenantPath, dtcsi.DatastorageDir))

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(memFs, filepath.Join(dtcsi.DataPath, dkTenantMapping))

		assert.NoError(t, err)
		assert.True(t, exists)

		data, err := afero.ReadFile(memFs, filepath.Join(dtcsi.DataPath, dkTenantMapping))

		assert.NoError(t, err)
		assert.Equal(t, tenantUUID, string(data))
	})
	t.Run(`error reading install version`, func(t *testing.T) {
		errorFs := &readVersionFileErrorFs{
			Fs: afero.NewMemMapFs(),
		}
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
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
			fs: errorFs,
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.EqualError(t, err, "failed to query installed OneAgent version: "+errorMsg)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)

		tenantPath := filepath.Join(dtcsi.DataPath, tenantUUID)
		exists, err := afero.Exists(errorFs, tenantPath)

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(errorFs, filepath.Join(tenantPath, dtcsi.LogDir))

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(errorFs, filepath.Join(tenantPath, dtcsi.DatastorageDir))

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(errorFs, filepath.Join(dtcsi.DataPath, dkTenantMapping))

		assert.NoError(t, err)
		assert.True(t, exists)

		data, err := afero.ReadFile(errorFs, filepath.Join(dtcsi.DataPath, dkTenantMapping))

		assert.NoError(t, err)
		assert.Equal(t, tenantUUID, string(data))
	})
	t.Run(`correct directories are created`, func(t *testing.T) {
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
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
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
		}

		err := afero.WriteFile(memFs, filepath.Join(dtcsi.DataPath, tenantUUID, dtcsi.VersionDir), []byte(agentVersion), fs.FileMode(0755))

		require.NoError(t, err)

		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 5 * time.Minute}, result)

		for _, path := range []string{
			filepath.Join(dtcsi.DataPath, tenantUUID),
			filepath.Join(dtcsi.DataPath, tenantUUID, dtcsi.LogDir),
			filepath.Join(dtcsi.DataPath, tenantUUID, dtcsi.DatastorageDir),
		} {
			exists, err := afero.Exists(memFs, path)

			assert.NoError(t, err)
			assert.True(t, exists)

			fileInfo, err := memFs.Stat(path)

			assert.NoError(t, err)
			assert.True(t, fileInfo.IsDir())
		}
	})
}
