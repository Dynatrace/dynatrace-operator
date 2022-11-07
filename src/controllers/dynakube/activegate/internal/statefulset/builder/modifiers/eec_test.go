package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestExtensionController_BuildContainerAndVolumes(t *testing.T) {
	assertion := assert.New(t)
	requirement := require.New(t)

	t.Run("happy path", func(t *testing.T) {
		dynakube := getBaseDynakube()
		eec := NewExtensionControllerModifier(dynakube, capability.NewMultiCapability(&dynakube))
		container := eec.buildContainer()

		assertion.NotEmpty(container.ReadinessProbe, "Expected readiness probe is defined")
		assertion.Equal("/readyz", container.ReadinessProbe.HTTPGet.Path, "Expected there is a readiness probe at /readyz")
		assertion.Empty(container.LivenessProbe, "Expected there is no liveness probe (not implemented)")
		assertion.Empty(container.StartupProbe, "Expected there is no startup probe")

		for _, port := range []int32{eecIngestPort} {
			assertion.Truef(kubeobjects.PortIsIn(container.Ports, port), "Expected that EEC container defines port %d", port)
		}

		for _, mountPath := range []string{
			consts.GatewayConfigMountPoint,
			dataSourceStartupArgsMountPoint,
			dataSourceAuthTokenMountPoint,
			statsdMetadataMountPoint,
			extensionsLogsDir,
			statsdLogsDir,
		} {
			assertion.Truef(kubeobjects.MountPathIsIn(container.VolumeMounts, mountPath), "Expected that EEC container defines mount point %s", mountPath)
		}

		assert.Truef(t, kubeobjects.MountPathIsReadOnlyOrReadWrite(container.VolumeMounts, consts.GatewayConfigMountPoint, kubeobjects.ReadOnlyMountPath),
			"Expected that ActiveGate container mount point %s is mounted ReadOnly", consts.GatewayConfigMountPoint,
		)

		for _, envVar := range []string{
			envTenantId, envServerUrl, envEecIngestPort,
		} {
			assertion.Truef(kubeobjects.EnvVarIsIn(container.Env, envVar), "Expected that EEC container defined environment variable %s", envVar)
		}

		// Logging a newline because otherwise `go test` doesn't recognise the result
		logger.Factory.GetLogger("extension controller").Info("")
	})

	t.Run("hardened container security context", func(t *testing.T) {
		dynakube := getBaseDynakube()
		eec := NewExtensionControllerModifier(dynakube, capability.NewMultiCapability(&dynakube))
		container := eec.buildContainer()

		requirement.NotNil(container.SecurityContext)
		securityContext := container.SecurityContext

		assertion.False(*securityContext.Privileged)
		assertion.False(*securityContext.AllowPrivilegeEscalation)
		assertion.False(*securityContext.ReadOnlyRootFilesystem)

		// Logging a newline because otherwise `go test` doesn't recognise the result
		logger.Factory.GetLogger("extension controller").Info("")
	})

	t.Run("resource requirements from feature flags", func(t *testing.T) {
		dynakube := getBaseDynakube()
		dynakube.ObjectMeta.Annotations[dynatracev1beta1.AnnotationFeaturePrefix+"activegate-eec-resources-limits-cpu"] = "200m"
		eec := NewExtensionControllerModifier(dynakube, capability.NewMultiCapability(&dynakube))
		container := eec.buildContainer()

		require.Empty(t, container.Resources.Requests)
		require.NotEmpty(t, container.Resources.Limits)

		assert.Equal(t, resource.NewScaledQuantity(200, resource.Milli).String(), container.Resources.Limits.Cpu().String())
		assert.True(t, container.Resources.Limits.Memory().IsZero())

		// Logging a newline because otherwise `go test` doesn't recognise the result
		logger.Factory.GetLogger("extension controller").Info("")
	})
}

func TestBuildEecConfigMapName(t *testing.T) {
	t.Run("happy case", func(t *testing.T) {
		eecConfigMapName := capability.BuildEecConfigMapName("dynakube", "activegate")
		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMapName)
	})

	t.Run("happy case, capitalized and with spaces", func(t *testing.T) {
		eecConfigMapName := capability.BuildEecConfigMapName("DynaKube", "Active Gate")
		assert.Equal(t, "DynaKube-Active_Gate-eec-config", eecConfigMapName)
	})

	t.Run("empty module", func(t *testing.T) {
		eecConfigMapName := capability.BuildEecConfigMapName("DynaKube", "")
		assert.Equal(t, "DynaKube--eec-config", eecConfigMapName)
	})

	t.Run("whitespace-only module", func(t *testing.T) {
		eecConfigMapName := capability.BuildEecConfigMapName("DynaKube", " 		")
		assert.Equal(t, "DynaKube-___-eec-config", eecConfigMapName)
	})
}
