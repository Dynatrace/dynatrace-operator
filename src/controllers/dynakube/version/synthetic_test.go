package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticUseTenantRegistry(t *testing.T) {
	testVersion := "1.2.3"
	testHash := getTestDigest()
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
		mockClient := &dtclient.MockDynatraceClient{}
		registry := newFakeRegistry(map[string]registry.ImageVersion{
			expectedImage: {
				Version: testVersion,
				Digest:  testHash,
			},
		})
		updater := newSyntheticUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO(), "")
		require.NoError(t, err, "default image set")
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.Synthetic.VersionStatus)
	})
}
