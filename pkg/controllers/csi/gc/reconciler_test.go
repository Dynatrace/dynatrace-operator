package csigc

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
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
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiUrl,
			},
		}
		gc := CSIGarbageCollector{
			apiReader: fake.NewClient(&dk),
			fs:        afero.NewMemMapFs(),
			db:        metadata.FakeMemoryDB(),
		}
		result, err := gc.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: dk.Name}})

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
	})
}
