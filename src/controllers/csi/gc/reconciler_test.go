package csigc

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCollectGCInfo(t *testing.T) {
	tenantUUID := "testTenant"
	apiUrl := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID)
	namespace := "test-namespace"
	latestVersion := "test-version"
	codeModulesImage := "test:tag"

	t.Run(`no pinned version`, func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: latestVersion,
			},
		}
		dkList := dynatracev1beta1.DynaKubeList{
			Items: []dynatracev1beta1.DynaKube{
				dynakube,
			},
		}

		gcInfo, err := collectGCInfo(dynakube, &dkList)
		require.NoError(t, err)
		assert.Equal(t, tenantUUID, gcInfo.tenantUUID)
		assert.Equal(t, latestVersion, gcInfo.latestAgentVersion)
		assert.Empty(t, gcInfo.pinnedVersions)
	})
	t.Run(`1 pinned version`, func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: codeModulesImage,
						},
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: latestVersion,
			},
		}
		dkList := dynatracev1beta1.DynaKubeList{
			Items: []dynatracev1beta1.DynaKube{
				dynakube,
			},
		}

		gcInfo, err := collectGCInfo(dynakube, &dkList)
		require.NoError(t, err)
		assert.Len(t, gcInfo.pinnedVersions, 1)
	})
	t.Run(`multi pinned version`, func(t *testing.T) {
		cloudNativeDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test1",
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: codeModulesImage,
						},
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: latestVersion,
			},
		}
		appMonitoringDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test2",
				Namespace: namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: apiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						Version: latestVersion,
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				LatestAgentVersionUnixPaas: latestVersion,
			},
		}
		dkList := dynatracev1beta1.DynaKubeList{
			Items: []dynatracev1beta1.DynaKube{
				cloudNativeDynakube,
				appMonitoringDynakube,
			},
		}

		gcInfo, err := collectGCInfo(cloudNativeDynakube, &dkList)
		require.NoError(t, err)
		assert.Len(t, gcInfo.pinnedVersions, 2)
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
		codeModulesImage := "test:tag"
		cloudNativeDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test1",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: codeModulesImage,
						},
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
		codeModulesRegistry := "test"
		cloudNativeDynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test1",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: codeModulesRegistry + ":" + codeModulesTag,
						},
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
