package version

import (
	"context"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticUseTenantRegistry(t *testing.T) {
	testVersion := "1.2.3"
	testHash := getTestDigest()
	t.Run("default image specified", func(t *testing.T) {
		dynakube := &dynatracev1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1.AnnotationFeatureSyntheticLocationEntityId: "non-existent",
				},
			},
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
			Status: dynatracev1.DynaKubeStatus{
				Synthetic: dynatracev1.SyntheticStatus{
					VersionStatus: dynatracev1.VersionStatus{
						Version: "non-empty",
					},
				},
			},
		}
		expectedImage := dynakube.DefaultSyntheticImage()
		mockClient := &dtclient.MockDynatraceClient{}
		registry := newFakeRegistry(map[string]ImageVersion{
			expectedImage: {
				Version: testVersion,
				Digest:  testHash,
			},
		})
		updater := newSyntheticUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO(), &dockerconfig.DockerConfig{})
		require.NoError(t, err, "default image set")
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.Synthetic.VersionStatus)
	})
}
