package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddInitArgs(t *testing.T) {
	t.Run("WithTechnologyAnnotation", func(t *testing.T) {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					oacommon.AnnotationTechnologies: "java",
				},
			},
		}
		initContainer := &corev1.Container{}
		dk := dynakube.DynaKube{}

		addInitArgs(pod, initContainer, dk, oacommon.DefaultInstallPath)

		require.Contains(t, initContainer.Args, "--technology=java")
	})

	t.Run("WithTechnologyFeature", func(t *testing.T) {
		pod := corev1.Pod{}
		initContainer := &corev1.Container{}
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynakube.AnnotationFeatureRemoteImageDownloadTechnology: "nodejs",
				},
			},
		}

		addInitArgs(pod, initContainer, dk, oacommon.DefaultInstallPath)

		require.Contains(t, initContainer.Args, "--technology=nodejs")
	})

	t.Run("WithoutTechnology", func(t *testing.T) {
		pod := corev1.Pod{}
		initContainer := &corev1.Container{}
		dk := dynakube.DynaKube{}

		addInitArgs(pod, initContainer, dk, oacommon.DefaultInstallPath)

		require.NotContains(t, initContainer.Args, "--technology=")
	})

	t.Run("WithDefaultArgs", func(t *testing.T) {
		pod := corev1.Pod{}
		initContainer := &corev1.Container{}
		dk := dynakube.DynaKube{}

		addInitArgs(pod, initContainer, dk, oacommon.DefaultInstallPath)

		require.Contains(t, initContainer.Args, "--source=/opt/dynatrace/oneagent")
		require.Contains(t, initContainer.Args, "--target=/mnt/bin")
	})
}
