package csigc

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	tenantUUID := "testTenant"
	apiUrl := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID)
	namespace := "test-namespace"

	t.Run(`no latest version in status`, func(t *testing.T) {
		dynakube := dynatracev1beta2.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
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
		assert.Equal(t, reconcile.Result{}, result)
	})
}

// mockIsNotMounted is rather confusing because of the double negation.
// you can pass in a map of filepaths, each path will be considered as mounted if corresponding error value is nil. (so returns false)
// if the filepath was not provided in the map, then the path is considered as not mounted. (so returns true)
// if an error was provided for a filepath in the map, then that path will cause the return of that error.
func mockIsNotMounted(files map[string]error) mountChecker {
	return func(mounter mount.Interface, file string) (bool, error) {
		err, ok := files[file]
		if !ok {
			return true, nil // unknown path => not mounted, no mocked error
		}

		if err == nil {
			return false, nil // known path => mounted, no mocked error
		}

		return false, err // mocked error for path
	}
}
