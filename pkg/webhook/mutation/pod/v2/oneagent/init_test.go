package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
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
					exp.OANodeImagePullTechnologiesKey: "nodejs",
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

	t.Run("WithFullstack", func(t *testing.T) {
		pod := corev1.Pod{}
		initContainer := &corev1.Container{}
		tenantUUID := "test-tenant-uuid"
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: tenantUUID,
						},
					},
				},
			},
		}

		addInitArgs(pod, initContainer, dk, oacommon.DefaultInstallPath)

		require.Contains(t, initContainer.Args, "--fullstack")
		require.Contains(t, initContainer.Args, "--tenant="+tenantUUID)
	})
}
