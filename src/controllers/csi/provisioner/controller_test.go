package csiprovisioner

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testAPIURL   = "http://test-uid/api"
	tenantUUID   = "test-uid"
	dkName       = "dynakube-test"
	otherDkName  = "other-dk"
	errorMsg     = "test-error"
	agentVersion = "12345"
	testZip      = `UEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAAIABwAdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADAOa5SAAAAAAAAAAAAAAAABQAcAHRlc3QvVVQJAAMXB55gHQeeYHV4CwABBOgDAAAE6AMAAFBLAwQKAAAAAACodKdSbC0hZxkAAAAZAAAADQAcAHRlc3QvdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAcAHRlc3QvdGVzdC9VVAkAAxwHnmAgB55gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAASABwAdGVzdC90ZXN0L3Rlc3QudHh0VVQJAAN8NJVgHAeeYHV4CwABBOgDAAAE6AMAAHlvdSBmb3VuZCB0aGUgZWFzdGVyIGVnZwpQSwMECgAAAAAA2zquUgAAAAAAAAAAAAAAAAYAHABhZ2VudC9VVAkAAy4JnmAxCZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAOI6rlIAAAAAAAAAAAAAAAALABwAYWdlbnQvY29uZi9VVAkAAzgJnmA+CZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAATABwAYWdlbnQvY29uZi90ZXN0LnR4dFVUCQADfDSVYDgJnmB1eAsAAQToAwAABOgDAAB5b3UgZm91bmQgdGhlIGVhc3RlciBlZ2cKUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAAgAGAAAAAAAAQAAAKSBAAAAAHRlc3QudHh0VVQFAAN8NJVgdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAwDmuUgAAAAAAAAAAAAAAAAUAGAAAAAAAAAAQAO1BWwAAAHRlc3QvVVQFAAMXB55gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAA0AGAAAAAAAAQAAAKSBmgAAAHRlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAYAAAAAAAAABAA7UH6AAAAdGVzdC90ZXN0L1VUBQADHAeeYHV4CwABBOgDAAAE6AMAAFBLAQIeAwoAAAAAAKh0p1JsLSFnGQAAABkAAAASABgAAAAAAAEAAACkgT4BAAB0ZXN0L3Rlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADbOq5SAAAAAAAAAAAAAAAABgAYAAAAAAAAABAA7UGjAQAAYWdlbnQvVVQFAAMuCZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAA4jquUgAAAAAAAAAAAAAAAAsAGAAAAAAAAAAQAO1B4wEAAGFnZW50L2NvbmYvVVQFAAM4CZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAABMAGAAAAAAAAQAAAKSBKAIAAGFnZW50L2NvbmYvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwUGAAAAAAgACACKAgAAjgIAAAAA`
)

type CSIGarbageCollectorMock struct {
	mock.Mock
}

func (m *CSIGarbageCollectorMock) Reconcile(context.Context, reconcile.Request) (reconcile.Result, error) {
	args := m.Called()
	return args.Get(0).(reconcile.Result), args.Error(1)
}

type mkDirAllErrorFs struct {
	afero.Fs
}

func (fs *mkDirAllErrorFs) MkdirAll(_ string, _ os.FileMode) error {
	return fmt.Errorf(errorMsg)
}

func TestOneAgentProvisioner_Reconcile(t *testing.T) {
	ctx := context.TODO()
	dynakubeName := "test-dk"

	t.Run(`no dynakube instance`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
		provisioner := &OneAgentProvisioner{
			apiReader: fake.NewClient(),
			db:        metadata.FakeMemoryDB(),
			gc:        gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{})

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`dynakube deleted`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
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
	t.Run(`application monitoring disabled`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
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

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)
	})
	t.Run(`csi driver not enabled`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
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

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)
	})
	t.Run(`csi driver disabled`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
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

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)

		dynakubeMetadatas, err := db.GetAllDynakubes(ctx)
		require.NoError(t, err)
		require.Len(t, dynakubeMetadatas, 0)
	})
	t.Run(`host monitoring used`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
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
							HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
						},
					},
				},
			),
			db: db,
			gc: gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakubeName}})

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: longRequeueDuration}, result)

		dynakubeMetadatas, err := db.GetAllDynakubes(ctx)
		require.NoError(t, err)
		require.Len(t, dynakubeMetadatas, 0)
	})
	t.Run(`no tokens`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
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
			),
			gc: gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.EqualError(t, err, `secrets "`+dkName+`" not found`)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when creating dynatrace client`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			Err: fmt.Errorf(errorMsg),
		}
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
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.DynatraceApiToken: []byte("test-value"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			gc:                     gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.EqualError(t, err, "failed to create Dynatrace client: "+errorMsg)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error creating directories`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gcError := errors.New("Custom GC error")
		gc.On("Create").Return(reconcile.Result{}, gcError)
		errorfs := &mkDirAllErrorFs{
			Fs: afero.NewMemMapFs(),
		}
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetOneAgentConnectionInfo").Return(dtclient.OneAgentConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: tenantUUID,
			},
		}, nil)
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			DynatraceClient: mockClient,
		}
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
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.DynatraceApiToken: []byte("api-token"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			fs:                     errorfs,
			db:                     metadata.FakeMemoryDB(),
			gc:                     gc,
		}
		result, err := provisioner.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.EqualError(t, err, "failed to create directory "+filepath.Join(tenantUUID)+": "+errorMsg)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{}, result)

		// Logging newline so go test can parse the output correctly
		log.Info("")
	})
	t.Run(`error getting latest agent version`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Reconcile").Return(reconcile.Result{}, nil)
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetOneAgentConnectionInfo").Return(dtclient.OneAgentConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: tenantUUID,
			},
		}, nil)
		mockClient.On("GetAgent",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("[]string"),
			mock.AnythingOfType("*mem.File")).Return(fmt.Errorf(errorMsg))
		mockClient.
			On("GetAgentVersions", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.Flavor, mock.AnythingOfType("string")).
			Return(make([]string, 0), fmt.Errorf(errorMsg))
		mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("uint")).Return(&testProcessModuleConfig, nil)
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			DynatraceClient: mockClient,
		}
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
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.DynatraceApiToken: []byte("api-token"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			fs:                     memFs,
			db:                     metadata.FakeMemoryDB(),
			recorder:               &record.FakeRecorder{},
			gc:                     gc,
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
	t.Run(`error getting dynakube from db`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Create").Return(reconcile.Result{}, nil)
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetOneAgentConnectionInfo").Return(dtclient.OneAgentConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: tenantUUID,
			},
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(agentVersion, nil)
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			DynatraceClient: mockClient,
		}
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
				&v1.Secret{
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

		gc.AssertNumberOfCalls(t, "Create", 0)
		require.Error(t, err)
		require.Empty(t, result)
	})
	t.Run(`correct directories are created`, func(t *testing.T) {
		gc := &CSIGarbageCollectorMock{}
		gc.On("Reconcile").Return(reconcile.Result{}, nil)
		memFs := afero.NewMemMapFs()
		memDB := metadata.FakeMemoryDB()
		err := memDB.InsertDynakube(ctx, metadata.NewDynakube(dkName, tenantUUID, agentVersion, "", 0))
		require.NoError(t, err)

		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetOneAgentConnectionInfo").Return(dtclient.OneAgentConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: tenantUUID,
			},
		}, nil)
		mockClient.On("GetLatestAgentVersion",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(agentVersion, nil)
		mockClient.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.Flavor,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(6).(io.Writer)

				zipFile := setupTestZip(t, memFs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("uint")).Return(&testProcessModuleConfig, nil)
		mockDtcBuilder := &dynatraceclient.StubBuilder{
			DynatraceClient: mockClient,
		}
		r := &OneAgentProvisioner{
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
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: dkName,
					},
					Data: map[string][]byte{
						dtclient.DynatraceApiToken: []byte("api-token"),
					},
				},
			),
			dynatraceClientBuilder: mockDtcBuilder,
			fs:                     memFs,
			db:                     memDB,
			recorder:               &record.FakeRecorder{},
			gc:                     gc,
		}

		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName}})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, reconcile.Result{RequeueAfter: 5 * time.Minute}, result)

		exists, err := afero.Exists(memFs, tenantUUID)

		require.NoError(t, err)
		require.True(t, exists)

		fileInfo, err := memFs.Stat(tenantUUID)

		require.NoError(t, err)
		require.True(t, fileInfo.IsDir())
	})
}

func TestHasCodeModulesWithCSIVolumeEnabled(t *testing.T) {
	t.Run(`default DynaKube object returns false`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{}

		isEnabled := dk.NeedsCSIDriver()

		require.False(t, isEnabled)
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

		require.True(t, isEnabled)
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

func setupTestZip(t *testing.T, fs afero.Fs) afero.File {
	zipf, err := base64.StdEncoding.DecodeString(testZip)
	require.NoError(t, err)

	zipFile, err := afero.TempFile(fs, "", "")
	require.NoError(t, err)

	_, err = zipFile.Write(zipf)
	require.NoError(t, err)

	err = zipFile.Sync()
	require.NoError(t, err)

	_, err = zipFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	return zipFile
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
