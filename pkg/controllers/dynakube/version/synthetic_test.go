package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	mockedclient "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticUseTenantRegistry(t *testing.T) {
	t.Run("default image specified", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "non-existent",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				Synthetic: dynatracev1beta1.SyntheticStatus{
					VersionStatus: status.VersionStatus{
						Version: "non-empty",
					},
				},
			},
		}
		expectedImage := dynakube.DefaultSyntheticImage()
		mockClient := mockedclient.NewClient(t)

		updater := newSyntheticUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.TODO())
		require.NoError(t, err, "default image set")
		assertStatusBasedOnTenantRegistry(t, expectedImage, versionUnknown, dynakube.Status.Synthetic.VersionStatus)
	})
}
