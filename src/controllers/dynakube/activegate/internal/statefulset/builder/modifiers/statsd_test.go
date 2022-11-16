package modifiers

import (
	corev1 "k8s.io/api/core/v1"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestStatsd_BuildContainerAndVolumes(t *testing.T) {
	assertion := assert.New(t)
	requirement := require.New(t)

	t.Run("modifier creates correct container", func(t *testing.T) {
		dynakube := getBaseDynakube()
		statsd := NewStatsdModifier(dynakube, capability.NewMultiCapability(&dynakube))
		container := statsd.buildContainer()

		assertIsContainerCorrect(assertion, container)

		dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities,
			dynatracev1beta1.RoutingCapability.DisplayName)
		statsd = NewStatsdModifier(dynakube, capability.NewMultiCapability(&dynakube))
		container = statsd.buildContainer()

		assertIsContainerCorrect(assertion, container)
	})

	t.Run("hardened container security context", func(t *testing.T) {
		dynakube := getBaseDynakube()
		statsd := NewStatsdModifier(dynakube, capability.NewMultiCapability(&dynakube))
		container := statsd.buildContainer()

		requirement.NotNil(container.SecurityContext)
		securityContext := container.SecurityContext

		assertion.False(*securityContext.Privileged)
		assertion.False(*securityContext.AllowPrivilegeEscalation)
		assertion.True(*securityContext.ReadOnlyRootFilesystem)

		assertion.True(*securityContext.RunAsNonRoot)
		assertion.Equal(kubeobjects.UnprivilegedUser, *securityContext.RunAsUser)
		assertion.Equal(kubeobjects.UnprivilegedGroup, *securityContext.RunAsGroup)
	})
	t.Run("resource requirements from feature flags", func(t *testing.T) {
		dynakube := getBaseDynakube()
		dynakube.ObjectMeta.Annotations[dynatracev1beta1.AnnotationFeaturePrefix+"activegate-statsd-resources-requests-memory"] = "500M"
		statsd := NewStatsdModifier(dynakube, capability.NewMultiCapability(&dynakube))

		container := statsd.buildContainer()

		require.NotEmpty(t, container.Resources.Requests)
		require.Empty(t, container.Resources.Limits)

		assert.True(t, container.Resources.Requests.Cpu().IsZero())
		assert.Equal(t, resource.NewScaledQuantity(500, resource.Mega).String(), container.Resources.Requests.Memory().String())
	})
}

func assertIsContainerCorrect(assertion *assert.Assertions, container corev1.Container) {
	assertion.NotEmpty(container.ReadinessProbe, "Expected readiness probe is defined")
	assertion.Equal("/readyz", container.ReadinessProbe.HTTPGet.Path, "Expected there is a readiness probe at /readyz")
	assertion.NotEmpty(container.LivenessProbe, "Expected liveness probe is defined")
	assertion.Equal("/livez", container.LivenessProbe.HTTPGet.Path, "Expected there is a liveness probe at /livez")
	assertion.Empty(container.StartupProbe, "Expected there is no startup probe")

	expectedPorts := []int32{
		consts.StatsdIngestPort,
		statsdProbesPort,
	}
	expectedMountPaths := []string{
		dataSourceStartupArgsMountPoint,
		dataSourceAuthTokenMountPoint,
		dataSourceMetadataMountPoint,
		statsdLogsDir,
	}
	expectedEnvVars := []string{
		envStatsdStartupArgsPath,
		envDataSourceProbeServerPort,
		envStatsdMetadata,
		envDataSourceLogFile,
	}

	for _, expectedPort := range expectedPorts {
		assertion.Truef(kubeobjects.PortIsIn(container.Ports, expectedPort),
			"Expected that StatsD container defines port %d", expectedPort)
	}

	for _, expectedMountPath := range expectedMountPaths {
		assertion.Truef(kubeobjects.MountPathIsIn(container.VolumeMounts, expectedMountPath),
			"Expected that StatsD container defines mount point %s", expectedMountPath)
	}

	for _, expectedEnvVar := range expectedEnvVars {
		assertion.Truef(kubeobjects.EnvVarIsIn(container.Env, expectedEnvVar),
			"Expected that StatsD container defined environment variable %s", expectedEnvVar)
	}
}
