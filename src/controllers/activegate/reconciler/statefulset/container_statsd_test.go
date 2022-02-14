package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
)

func TestStatsd_BuildContainerAndVolumes(t *testing.T) {
	assertion := assert.New(t)

	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
		"", "", "test-feature", "", "",
		nil, nil, nil,
	)

	t.Run("happy path", func(t *testing.T) {
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
			"/mnt/dsexecargs",
			"/var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources",
			"/mnt/dsmetadata",
		} {
			assertion.Truef(kubeobjects.MountPathIsIn(container.VolumeMounts, mountPath), "Expected that StatsD container defines mount point %s", mountPath)
		}

		for _, envVar := range []string{
			"StatsdExecArgsPath", "ProbeServerPort", "StatsdMetadataDir",
		} {
			assertion.Truef(kubeobjects.EnvVarIsIn(container.Env, envVar), "Expected that StatsD container defined environment variable %s", envVar)
		}
	})

	t.Run("volumes vs volume mounts", func(t *testing.T) {
		eec := NewExtensionController(stsProperties)
		statsd := NewStatsd(stsProperties)
		volumes := buildVolumes(stsProperties, []kubeobjects.ContainerBuilder{eec, statsd})

		container := statsd.BuildContainer()
		for _, volumeMount := range container.VolumeMounts {
			assertion.Truef(kubeobjects.VolumeIsDefined(volumes, volumeMount.Name), "Expected that volume mount %s has a predefined pod volume", volumeMount.Name)
		}
	})
}
