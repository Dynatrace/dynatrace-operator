package version

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticUseTenantRegistry(t *testing.T) {
	testVersion := "1.2.3"
	testHash := "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
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
					VersionStatus: dynatracev1beta1.VersionStatus{
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
				Hash:    testHash,
			},
		})
		updater := newSyntheticUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO(), &dockerconfig.DockerConfig{})
		require.NoError(t, err, "default image set")
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.Synthetic.VersionStatus)
	})
}
