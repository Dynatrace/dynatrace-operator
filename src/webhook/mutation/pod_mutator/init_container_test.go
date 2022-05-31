package pod_mutator

import (
	"testing"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateInstallInitContainerBase(t *testing.T) {
	t.Run("should create the init container", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		podWebhook := createTestWebhook(t,
			[]dtwebhook.PodMutator{},
			[]client.Object{})
		initContainer := podWebhook.createInstallInitContainerBase(pod, dynakube)
		require.NotNil(t, initContainer)
		assert.Equal(t, initContainer.Image, podWebhook.webhookImage)
		assert.Equal(t, initContainer.Resources, testResourceRequirements)
		assert.Equal(t, initContainer.SecurityContext, testSecurityContext)
	})
}
