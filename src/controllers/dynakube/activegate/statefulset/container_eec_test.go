package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

func testBuildStsProperties() *statefulSetProperties {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	return NewStatefulSetProperties(instance, capabilityProperties,
		"", "", "test-feature", "", "",
		nil, nil, nil,
	)
}

func TestExtensionController_BuildContainerAndVolumes(t *testing.T) {
	assertion := assert.New(t)
	requirement := require.New(t)

	t.Run("happy path", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		eec := NewExtensionController(stsProperties)
		container := eec.BuildContainer()

		assertion.NotEmpty(container.ReadinessProbe, "Expected readiness probe is defined")
		assertion.Equal("/readyz", container.ReadinessProbe.HTTPGet.Path, "Expected there is a readiness probe at /readyz")
		assertion.Empty(container.LivenessProbe, "Expected there is no liveness probe (not implemented)")
		assertion.Empty(container.StartupProbe, "Expected there is no startup probe")

		for _, port := range []int32{eecIngestPort} {
			assertion.Truef(kubeobjects.PortIsIn(container.Ports, port), "Expected that EEC container defines port %d", port)
		}

		for _, mountPath := range []string{
			capability.ActiveGateGatewayConfigMountPoint,
			dataSourceStartupArgsMountPoint,
			dataSourceAuthTokenMountPoint,
			statsdMetadataMountPoint,
			extensionsLogsDir,
			statsdLogsDir,
		} {
			assertion.Truef(kubeobjects.MountPathIsIn(container.VolumeMounts, mountPath), "Expected that EEC container defines mount point %s", mountPath)
		}

		assert.Truef(t, kubeobjects.MountPathIsReadOnlyOrReadWrite(container.VolumeMounts, capability.ActiveGateGatewayConfigMountPoint, kubeobjects.ReadOnlyMountPath),
			"Expected that ActiveGate container mount point %s is mounted ReadOnly", capability.ActiveGateGatewayConfigMountPoint,
		)

		for _, envVar := range []string{
			envTenantId, envServerUrl, envEecIngestPort,
		} {
			assertion.Truef(kubeobjects.EnvVarIsIn(container.Env, envVar), "Expected that EEC container defined environment variable %s", envVar)
		}
	})

	t.Run("hardened container security context", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		container := NewExtensionController(stsProperties).BuildContainer()

		requirement.NotNil(container.SecurityContext)
		securityContext := container.SecurityContext

		assertion.False(*securityContext.Privileged)
		assertion.False(*securityContext.AllowPrivilegeEscalation)
		assertion.False(*securityContext.ReadOnlyRootFilesystem)
	})

	t.Run("volumes vs volume mounts", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		stsProperties.Spec.ActiveGate.Capabilities = append(stsProperties.Spec.ActiveGate.Capabilities, dynatracev1beta1.StatsdIngestCapability.DisplayName)
		eec := NewExtensionController(stsProperties)
		statsd := NewStatsd(stsProperties)
		volumes := buildVolumes(stsProperties, []kubeobjects.ContainerBuilder{eec, statsd})

		container := eec.BuildContainer()
		for _, volumeMount := range container.VolumeMounts {
			assertion.Truef(kubeobjects.VolumeIsDefined(volumes, volumeMount.Name), "Expected that volume mount %s has a predefined pod volume", volumeMount.Name)
		}
	})

	t.Run("resource requirements from feature flags", func(t *testing.T) {
		stsProperties := testBuildStsProperties()
		stsProperties.ObjectMeta.Annotations[dynatracev1beta1.AnnotationFeaturePrefix+"activegate-eec-resources-limits-cpu"] = "200m"
		eec := NewExtensionController(stsProperties)

		container := eec.BuildContainer()

		require.Empty(t, container.Resources.Requests)
		require.NotEmpty(t, container.Resources.Limits)

		assert.Equal(t, resource.NewScaledQuantity(200, resource.Milli).String(), container.Resources.Limits.Cpu().String())
		assert.True(t, container.Resources.Limits.Memory().IsZero())
	})
}

func TestBuildEecConfigMapName(t *testing.T) {
	t.Run("happy case", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("dynakube", "activegate")
		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMapName)
	})

	t.Run("happy case, capitalized and with spaces", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("DynaKube", "Active Gate")
		assert.Equal(t, "DynaKube-Active_Gate-eec-config", eecConfigMapName)
	})

	t.Run("empty module", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("DynaKube", "")
		assert.Equal(t, "DynaKube--eec-config", eecConfigMapName)
	})

	t.Run("whitespace-only module", func(t *testing.T) {
		eecConfigMapName := BuildEecConfigMapName("DynaKube", " 		")
		assert.Equal(t, "DynaKube-___-eec-config", eecConfigMapName)
	})
}
