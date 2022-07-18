package csigc

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCollectGCInfo(t *testing.T) {
	ctx := context.TODO()
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
		client := fake.NewClient(&dynakube)

		gcInfo, err := collectGCInfo(ctx, client, dynakube)
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
		client := fake.NewClient(&dynakube)

		gcInfo, err := collectGCInfo(ctx, client, dynakube)
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
		client := fake.NewClient(&cloudNativeDynakube, &appMonitoringDynakube)

		gcInfo, err := collectGCInfo(ctx, client, cloudNativeDynakube)
		require.NoError(t, err)
		assert.Len(t, gcInfo.pinnedVersions, 2)
	})
}

func TestIsSafeToGC(t *testing.T) {
	t.Run(`error db ==> not safe`, func(t *testing.T) {
		isSafe := isSafeToGC(&metadata.FakeFailDB{})
		require.False(t, isSafe)
	})
	t.Run(`1 LatestVersion not set ==> not safe`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		require.NoError(t, db.InsertDynakube(&metadata.Dynakube{
			Name:          "dk1",
			TenantUUID:    "t1",
			LatestVersion: "",
			ImageDigest:   "",
		}))
		require.NoError(t, db.InsertDynakube(&metadata.Dynakube{
			Name:          "dk2",
			TenantUUID:    "t2",
			LatestVersion: "v2",
			ImageDigest:   "d2",
		}))
		require.False(t, isSafeToGC(db))
	})
	t.Run(`all LatestVersion set ==> safe`, func(t *testing.T) {
		db := metadata.FakeMemoryDB()
		require.NoError(t, db.InsertDynakube(&metadata.Dynakube{
			Name:          "dk1",
			TenantUUID:    "t1",
			LatestVersion: "v1",
			ImageDigest:   "",
		}))
		require.NoError(t, db.InsertDynakube(&metadata.Dynakube{
			Name:          "dk2",
			TenantUUID:    "t2",
			LatestVersion: "v2",
			ImageDigest:   "d2",
		}))
		require.True(t, isSafeToGC(db))
	})
}
