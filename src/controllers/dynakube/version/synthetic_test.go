package version

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/src/oci/registry/mocks"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticUseTenantRegistry(t *testing.T) {
	testVersion := "1.2.3.4-5"
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

		mockImageGetter := mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: testVersion, Digest: testHash}, nil)

		updater := newSyntheticUpdater(dynakube, fake.NewClient(), mockClient, &mockImageGetter)

		err := updater.UseTenantRegistry(context.TODO())
		require.NoError(t, err, "default image set")
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.Synthetic.VersionStatus)
	})
}
