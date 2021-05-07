package csiprovisioner

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testName    = "test-name"
	testError   = "test-error"
	testUid     = "test-uid"
	testVersion = "test-version"
)

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
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
			),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, `failed to query tokens: secrets "`+testName+`" not found`)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when creating dynatrace client`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return nil, fmt.Errorf(testError)
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to create Dynatrace client: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when querying dynatrace client for connection info`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{}, fmt.Errorf(testError))

		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to fetch connection info: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error creating directories`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: testUid,
		}, nil)
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			mkDirAllFunc: func(path string, perm fs.FileMode) error {
				return fmt.Errorf(testError)
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to create directory "+testUid+": "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error reading tenant file`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: testUid,
		}, nil)
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			mkDirAllFunc: func(path string, perm fs.FileMode) error {
				return nil
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return nil, fmt.Errorf(testError)
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to query assigned DynaKube tenant: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error getting latest agent version`, func(t *testing.T) {
		writtenFiles := make(map[string][]byte)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: testUid,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return("", fmt.Errorf(testError))
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			mkDirAllFunc: func(path string, perm fs.FileMode) error {
				return nil
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return nil, nil
			},
			writeFileFunc: func(filename string, data []byte, perm fs.FileMode) error {
				writtenFiles[filename] = data
				return nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.Contains(t, writtenFiles, "tenant-test-name")
		assert.Equal(t, testUid, string(writtenFiles["tenant-test-name"]))

		assert.EqualError(t, err, "failed to query OneAgent version: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error reading install version`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: testUid,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(testVersion, nil)
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			mkDirAllFunc: func(path string, perm fs.FileMode) error {
				return nil
			},
			readFileFunc: func(filename string) ([]byte, error) {
				if strings.HasSuffix(filename, versionDir) {
					return nil, fmt.Errorf(testError)
				}
				return nil, nil
			},
			writeFileFunc: func(filename string, data []byte, perm fs.FileMode) error {
				return nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to query installed OneAgent version: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`correct directories are created`, func(t *testing.T) {
		createdDirs := make(map[string]fs.FileMode)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			TenantUUID: testUid,
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(testVersion, nil)
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			mkDirAllFunc: func(path string, perm fs.FileMode) error {
				createdDirs[path] = perm
				return nil
			},
			readFileFunc: func(filename string) ([]byte, error) {
				if strings.HasSuffix(filename, versionDir) {
					return []byte(testVersion), nil
				}
				return nil, nil
			},
			writeFileFunc: func(filename string, data []byte, perm fs.FileMode) error {
				return nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.Contains(t, createdDirs, testUid)
		assert.Contains(t, createdDirs, filepath.Join(testUid, logDir))
		assert.Contains(t, createdDirs, filepath.Join(testUid, dataDir))

		assert.Equal(t, fs.FileMode(0755), createdDirs[testUid])
		assert.Equal(t, fs.FileMode(0755), createdDirs[filepath.Join(testUid, logDir)])
		assert.Equal(t, fs.FileMode(0755), createdDirs[filepath.Join(testUid, dataDir)])

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 5 * time.Minute}, result)
	})
}
