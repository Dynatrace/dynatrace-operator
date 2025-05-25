package metadata

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddMetadataToInitEnv(t *testing.T) {
	t.Run("check that annotations copied from pod to env", func(t *testing.T) {
		expectedKeys := []string{
			"beep",
			"boop",
			"hello",
		}

		request := createTestMutationRequest(nil, nil, false)
		request.Pod.Annotations = map[string]string{
			"test-annotation": "boom",
		}
		for _, key := range expectedKeys {
			request.Pod.Annotations[dynakube.MetadataPrefix+key] = key + "-value"
		}

		setMetadataAnnotationValue(request.Pod, request.Pod.Annotations)
		addMetadataToInitEnv(request.Pod, request.InstallContainer)

		annotationsEnv := env.FindEnvVar(request.InstallContainer.Env, consts.EnrichmentWorkloadAnnotationsEnv)
		require.NotNil(t, annotationsEnv)
		propagatedAnnotations := map[string]string{}
		err := json.Unmarshal([]byte(annotationsEnv.Value), &propagatedAnnotations)
		require.NoError(t, err)

		assert.Equal(t, len(expectedKeys)+1, len(propagatedAnnotations))

		for _, key := range expectedKeys {
			require.Contains(t, propagatedAnnotations, key)
			assert.Equal(t, key+"-value", propagatedAnnotations[key])
		}
	})
}

func setMetadataAnnotationValue(pod *corev1.Pod, annotations map[string]string) {
	metadataAnnotations := map[string]string{}
	for key, value := range annotations {
		// Annotations added to the json must not have metadata.dynatrace.com/ prefix
		if strings.HasPrefix(key, dynakube.MetadataPrefix) {
			split := strings.Split(key, dynakube.MetadataPrefix)
			metadataAnnotations[split[1]] = value
		} else {
			metadataAnnotations[key] = value
		}
	}

	marshaledAnnotations, err := json.Marshal(metadataAnnotations)
	if err != nil {
		log.Error(err, "failed to marshal annotations to map", "annotations", annotations)
	}

	pod.Annotations[dynakube.MetadataAnnotation] = string(marshaledAnnotations)
}
