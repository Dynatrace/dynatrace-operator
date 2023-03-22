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
