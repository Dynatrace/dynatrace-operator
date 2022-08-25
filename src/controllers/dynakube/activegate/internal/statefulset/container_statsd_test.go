package statefulset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestStatsd_BuildContainerAndVolumes(t *testing.T) {
	assertion := assert.New(t)
	requirement := require.New(t)

	t.Run("happy path", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		statsd := NewStatsd(stsProperties)
		container := statsd.BuildContainer()

		assertion.NotEmpty(container.ReadinessProbe, "Expected readiness probe is defined")
		assertion.Equal("/readyz", container.ReadinessProbe.HTTPGet.Path, "Expected there is a readiness probe at /readyz")
		assertion.NotEmpty(container.LivenessProbe, "Expected liveness probe is defined")
		assertion.Equal("/livez", container.LivenessProbe.HTTPGet.Path, "Expected there is a liveness probe at /livez")
		assertion.Empty(container.StartupProbe, "Expected there is no startup probe")

		for _, port := range []int32{
			consts.StatsdIngestPort, statsdProbesPort,
		} {
			assertion.Truef(kubeobjects.PortIsIn(container.Ports, port), "Expected that StatsD container defines port %d", port)
		}

		for _, mountPath := range []string{
			dataSourceStartupArgsMountPoint,
			dataSourceAuthTokenMountPoint,
			dataSourceMetadataMountPoint,
			statsdLogsDir,
		} {
			assertion.Truef(kubeobjects.MountPathIsIn(container.VolumeMounts, mountPath), "Expected that StatsD container defines mount point %s", mountPath)
		}

		for _, envVar := range []string{
			envStatsdStartupArgsPath, envDataSourceProbeServerPort, envStatsdMetadata, envDataSourceLogFile,
		} {
			assertion.Truef(kubeobjects.EnvVarIsIn(container.Env, envVar), "Expected that StatsD container defined environment variable %s", envVar)
		}
	})

	t.Run("hardened container security context", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		container := NewStatsd(stsProperties).BuildContainer()

		requirement.NotNil(container.SecurityContext)
		securityContext := container.SecurityContext

		assertion.False(*securityContext.Privileged)
		assertion.False(*securityContext.AllowPrivilegeEscalation)
		assertion.True(*securityContext.ReadOnlyRootFilesystem)

		assertion.True(*securityContext.RunAsNonRoot)
		assertion.Equal(kubeobjects.UnprivilegedUser, *securityContext.RunAsUser)
		assertion.Equal(kubeobjects.UnprivilegedGroup, *securityContext.RunAsGroup)
	})

	t.Run("volumes vs volume mounts", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		eec := NewExtensionController(stsProperties)
		statsd := NewStatsd(stsProperties)
		volumes := buildVolumes(stsProperties, []kubeobjects.ContainerBuilder{eec, statsd})

		container := statsd.BuildContainer()
		for _, volumeMount := range container.VolumeMounts {
			assertion.Truef(kubeobjects.VolumeIsDefined(volumes, volumeMount.Name), "Expected that volume mount %s has a predefined pod volume", volumeMount.Name)
		}
	})

	t.Run("resource requirements from feature flags", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		stsProperties.ObjectMeta.Annotations[dynatracev1beta1.AnnotationFeaturePrefix+"activegate-statsd-resources-requests-memory"] = "500M"
		statsd := NewStatsd(stsProperties)

		container := statsd.BuildContainer()

		require.NotEmpty(t, container.Resources.Requests)
		require.Empty(t, container.Resources.Limits)

		assert.True(t, container.Resources.Requests.Cpu().IsZero())
		assert.Equal(t, resource.NewScaledQuantity(500, resource.Mega).String(), container.Resources.Requests.Memory().String())
	})
}
