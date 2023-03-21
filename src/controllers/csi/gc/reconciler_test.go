package csigc

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	tenantUUID := "testTenant"
	apiUrl := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID)
	namespace := "test-namespace"
	t.Run(`no latest version in status`, func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
			},
		}
		gc := CSIGarbageCollector{
			apiReader: fake.NewClient(&dynakube),
			fs:        afero.NewMemMapFs(),
			db:        metadata.FakeMemoryDB(),
		}
		result, err := gc.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dynakube.Name}})

		require.NoError(t, err)
		assert.Equal(t, result, reconcile.Result{})
	})
}

func TestCollectGCInfo(t *testing.T) {
	tenantUUID := "testTenant"
	apiUrl := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID)
	namespace := "test-namespace"
	latestVersion := "test-version"
	imageTag := "tag"

	t.Run(`1 pinned version`, func(t *testing.T) {
		oldVersion := "old-version"
		newVersionDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				CodeModules: dynatracev1beta1.CodeModulesStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						Version: latestVersion,
					},
				},
			},
		}

		oldVersionDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				CodeModules: dynatracev1beta1.CodeModulesStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						Version: oldVersion,
					},
				},
			},
		}
		dkList := dynatracev1beta1.DynaKubeList{
			Items: []dynatracev1beta1.DynaKube{
				newVersionDynakube,
				oldVersionDynakube,
			},
		}

		gcInfo := collectGCInfo(newVersionDynakube, &dkList)
		assert.Len(t, gcInfo.pinnedVersions, 2)
	})
	t.Run(`only consider version, not the tag`, func(t *testing.T) {
		versionDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test1",
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				CodeModules: dynatracev1beta1.CodeModulesStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						Version: latestVersion,
					},
				},
			},
		}
		tagDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test2",
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				CodeModules: dynatracev1beta1.CodeModulesStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						ImageTag: imageTag,
					},
				},
			},
		}
		dkList := dynatracev1beta1.DynaKubeList{
			Items: []dynatracev1beta1.DynaKube{
				versionDynakube,
				tagDynakube,
			},
		}

		gcInfo := collectGCInfo(versionDynakube, &dkList)
		assert.Len(t, gcInfo.pinnedVersions, 1)
		assert.True(t, gcInfo.pinnedVersions[latestVersion])
	})
}

func TestIsSafeToGC(t *testing.T) {
	ctx := context.TODO()
	t.Run(`error db ==> not safe`, func(t *testing.T) {
		isSafe := isSafeToGC(ctx, &metadata.FakeFailDB{}, nil)
		require.False(t, isSafe)
	})
	t.Run(`1 LatestVersion not set ==> not safe`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "dk1",
			TenantUUID:    "t1",
			LatestVersion: "",
			ImageDigest:   "",
		}))
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "dk2",
			TenantUUID:    "t2",
			LatestVersion: "v2",
			ImageDigest:   "d2",
		}))
		require.False(t, isSafeToGC(ctx, db, &dynatracev1beta1.DynaKubeList{}))
	})
	t.Run(`all LatestVersion set (no codemodules image) ==> safe`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "dk1",
			TenantUUID:    "t1",
			LatestVersion: "v1",
			ImageDigest:   "",
		}))
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "dk2",
			TenantUUID:    "t2",
			LatestVersion: "v2",
			ImageDigest:   "d2",
		}))
		require.True(t, isSafeToGC(ctx, db, &dynatracev1beta1.DynaKubeList{}))
	})
	t.Run(`LatestVersion doesn't match version in dynakube ==> not safe`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		imageTag := "v99"
		cloudNativeDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test1",
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				CodeModules: dynatracev1beta1.CodeModulesStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						ImageTag: imageTag,
					},
				},
			},
		}
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          cloudNativeDynakube.Name,
			TenantUUID:    "t1",
			LatestVersion: "v1",
			ImageDigest:   "",
		}))
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "dk2",
			TenantUUID:    "t2",
			LatestVersion: "v2",
			ImageDigest:   "d2",
		}))
		require.False(t, isSafeToGC(ctx, db, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{cloudNativeDynakube}}))
	})
	t.Run(`LatestVersion matches version in dynakube ==> safe`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		codeModulesTag := "tag"
		cloudNativeDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test1",
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				CodeModules: dynatracev1beta1.CodeModulesStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						ImageTag: codeModulesTag,
					},
				},
			},
		}
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          cloudNativeDynakube.Name,
			TenantUUID:    "t1",
			LatestVersion: codeModulesTag,
			ImageDigest:   "",
		}))
		require.NoError(t, db.InsertDynakube(ctx, &metadata.Dynakube{
			Name:          "dk2",
			TenantUUID:    "t2",
			LatestVersion: "v2",
			ImageDigest:   "d2",
		}))
		require.True(t, isSafeToGC(ctx, db, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{cloudNativeDynakube}}))
	})
}
