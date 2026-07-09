//go:build e2e

package csimigration

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8svolume"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	oneagentComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

// Feature verifies the CSI migration path:
//  1. Install with CSI active → sample app pods have CSI-mounted oneagent-bin volume.
//  2. Redeploy operator with csidriver.migrationMode=true → CSI DaemonSet stays running.
//  3. Restart sample app → existing CSI-mounted pods terminate cleanly (DaemonSet still present
//     to serve unmount requests), new pods are injected via emptyDir instead.
//  4. Verify new pods use emptyDir injection.
//  5. Redeploy operator with csidriver.enabled=false → operator comes up healthy without CSI.
func Feature(t *testing.T) features.Feature {
	builder := features.New("cloudnative-csi-migration")
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)

	sampleApp := sample.NewApp(t, &testDynakube, sample.AsDeployment())
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	builder.Assess("install sample app", sampleApp.Install())

	// Phase 1: verify CSI injection is active before migration.
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	builder.Assess("sample app has CSI volume", assertCSIVolume(sampleApp, assertHasCSIVolume))

	// Phase 2: switch operator to migration mode — operator removes osagent-storage CSI volume from OneAgent DaemonSet.
	builder.Assess("redeploy operator with migration mode", enableMigrationMode)
	// Must wait for DaemonSet rollout: ensures all OneAgent pods have dropped their CSI volumes before
	// Phase 5 removes the CSI DaemonSet entirely (avoids stuck Terminating pods on unmount).
	builder.Assess("oneagent daemonset rolled out after migration mode enabled", oneagentComponents.WaitForDaemonset(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("dynakube reconciled after migration mode enabled", dynakubeComponents.WaitForPhase(testDynakube, status.Running))

	// Phase 3: restart sample app — old CSI-mounted pods must terminate cleanly.
	builder.Assess("restart sample app", sampleApp.Restart())

	// Phase 4: verify new pods are injected but no longer use CSI volumes.
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	builder.Assess("sample app injected without CSI volume after migration", assertCSIVolume(sampleApp, assertHasNoCSIVolume))

	// Phase 5: disable CSI driver entirely — operator reconciles OneAgent DaemonSet from CSI to HostPath volumes.
	builder.Assess("redeploy operator with CSI disabled", disableCSIDriver)
	builder.Assess("oneagent daemonset ready after CSI disabled", oneagentComponents.WaitForDaemonset(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("dynakube reconciled after CSI disabled", dynakubeComponents.WaitForPhase(testDynakube, status.Running))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

func enableMigrationMode(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	require.NoError(t, operator.InstallViaHelm("", true, helm.WithArgs("--set", "csidriver.migrationMode=true")))
	ctx, err := operator.VerifyInstall(ctx, cfg, true)
	require.NoError(t, err)

	return ctx
}

func disableCSIDriver(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	require.NoError(t, operator.InstallViaHelm("", false))
	ctx, err := operator.VerifyInstall(ctx, cfg, false)
	require.NoError(t, err)

	return ctx
}

func assertCSIVolume(sampleApp *sample.App, check func(t *testing.T, vol *corev1.Volume, name string)) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.ListPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items)

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			vol := k8svolume.FindByName(pod.Spec.Volumes, oacommon.BinVolumeName)
			check(t, vol, pod.Name)
		}

		return ctx
	}
}

func assertHasCSIVolume(t *testing.T, vol *corev1.Volume, name string) {
	require.NotNilf(t, vol, "pod %s: volume %s not found", name, oacommon.BinVolumeName)
	require.NotNilf(t, vol.CSI, "pod %s: expected CSI volume for %s", name, oacommon.BinVolumeName)
	assert.Equal(t, dtcsi.DriverName, vol.CSI.Driver)
}

func assertHasNoCSIVolume(t *testing.T, vol *corev1.Volume, name string) {
	require.NotNilf(t, vol, "pod %s: volume %s not found after migration", name, oacommon.BinVolumeName)
	assert.NotNilf(t, vol.EmptyDir, "pod %s: expected emptyDir for %s after migration", name, oacommon.BinVolumeName)
	assert.Nilf(t, vol.CSI, "pod %s: volume %s still uses CSI after migration", name, oacommon.BinVolumeName)
}
