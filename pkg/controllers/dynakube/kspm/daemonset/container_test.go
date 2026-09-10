// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestGetContainer(t *testing.T) {
	t.Cleanup(version.DisableCacheForTest(123))

	tenant := "test-tenant"

	t.Run("get main container", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				KSPM: &kspm.Spec{},
			},
			Status: dynakube.DynaKubeStatus{
				KSPM: kspm.Status{ResolvedImage: "test-repo/test-image:test-tag"},
			},
		}
		mainContainer := getContainer(dk, tenant)

		require.NotEmpty(t, mainContainer)

		assert.NotEmpty(t, mainContainer.Name)
		assert.Equal(t, "test-repo/test-image:test-tag", mainContainer.Image)
		assert.Empty(t, mainContainer.ImagePullPolicy)
		assert.NotEmpty(t, mainContainer.VolumeMounts)
		assert.Len(t, mainContainer.VolumeMounts, expectedMountLen)
		assert.NotEmpty(t, mainContainer.Env)
		assert.Len(t, mainContainer.Env, expectedBaseEnvLen)
		assert.NotEmpty(t, mainContainer.SecurityContext)
		assert.NotEmpty(t, mainContainer.SecurityContext.SeccompProfile)
	})

	// the image itself comes from the status, see TestImageResolution
	t.Run("pull policy is taken from the image-ref", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				KSPM: &kspm.Spec{},
			},
		}
		dk.KSPM().ImageRef = image.Ref{
			Repository: "my-test-repo",
			Tag:        "my-test-tag",
			PullPolicy: corev1.PullAlways,
		}
		mainContainer := getContainer(dk, tenant)

		require.NotEmpty(t, mainContainer)
		assert.Equal(t, corev1.PullAlways, mainContainer.ImagePullPolicy)
	})
}
