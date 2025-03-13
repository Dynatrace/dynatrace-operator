package metadata

import (
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddMetadataToInitEnv(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from pod to env", func(t *testing.T) {
		expectedKeys := []string{
			"beep",
			"boop",
			"hello",
		}
		notExpectedKey := "no-prop"
		request := createTestMutationRequest(nil, nil, false)
		request.Pod.Labels = map[string]string{
			dynakube.MetadataPrefix + notExpectedKey: "beep-boop",
			"test-label":                             "boom",
		}
		request.Pod.Annotations = map[string]string{
			"test-annotation": "boom",
		}

		for _, key := range expectedKeys {
			request.Pod.Annotations[dynakube.MetadataPrefix+key] = key + "-value"
		}

		addMetadataToInitEnv(request.Pod, request.InstallContainer)

		annotationsEnv := env.FindEnvVar(request.InstallContainer.Env, consts.EnrichmentWorkloadAnnotationsEnv)
		require.NotNil(t, annotationsEnv)

		propagatedAnnotations := map[string]string{}
		err := json.Unmarshal([]byte(annotationsEnv.Value), &propagatedAnnotations)
		require.NoError(t, err)

		for _, key := range expectedKeys {
			require.Contains(t, propagatedAnnotations, key)
			assert.Equal(t, key+"-value", propagatedAnnotations[key])
			assert.NotContains(t, propagatedAnnotations, notExpectedKey)
		}
	})
}
