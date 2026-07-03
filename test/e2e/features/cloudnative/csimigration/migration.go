//go:build e2e

package csimigration

import (
	"context"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	csipkg "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/csi"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

// Feature verifies the CSI migration path:
//  1. Install with CSI active → sample app pods have CSI-mounted oneagent-bin volume.
//  2. Redeploy operator with csidriver.migrationMode=true → CSI DaemonSet stays running,
//  3. Restart sample app → existing CSI-mounted pods terminate cleanly (DaemonSet still present
//     to serve unmount requests), new pods are injected via emptyDir instead.
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
	builder.Assess("sample app has CSI volume", assertHasCSIVolume(sampleApp))

	// Phase 2: switch operator to migration mode.
	builder.Assess("redeploy operator with migration mode", helpers.ToFeatureFunc(enableMigrationMode(), true))
	builder.Assess("CSI DaemonSet still present after migration mode enabled", k8sdaemonset.IsReady(csipkg.DaemonSetName, operator.DefaultNamespace))

	// Phase 3: restart sample app — old CSI-mounted pods must terminate cleanly.
	builder.Assess("restart sample app", sampleApp.Restart())

	// Phase 4: verify new pods are injected but no longer use CSI volumes.
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	builder.Assess("sample app injected without CSI volume after migration", assertHasNoCSIVolume(sampleApp))

	builder.WithTeardown("restore operator without migration mode", helpers.ToFeatureFunc(disableMigrationMode(), true))
	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

func enableMigrationMode() env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		if err := operator.InstallViaHelm("", true, helm.WithArgs("--set", "csidriver.migrationMode=true")); err != nil {
			return ctx, err
		}

		return operator.VerifyInstall(ctx, envConfig, true)
	}
}

func disableMigrationMode() env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		if err := operator.InstallViaHelm("", true); err != nil {
			return ctx, err
		}

		return operator.VerifyInstall(ctx, envConfig, true)
	}
}

func assertHasCSIVolume(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.ListPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items)

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			found := false

			for _, volume := range pod.Spec.Volumes {
				if volume.Name == oacommon.BinVolumeName {
					found = true
					require.NotNilf(t, volume.CSI, "pod %s: expected CSI volume for %s", pod.Name, oacommon.BinVolumeName)
					assert.Equal(t, dtcsi.DriverName, volume.CSI.Driver)
				}
			}

			assert.Truef(t, found, "pod %s: volume %s not found", pod.Name, oacommon.BinVolumeName)
		}

		return ctx
	}
}

func assertHasNoCSIVolume(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.ListPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items)

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			for _, volume := range pod.Spec.Volumes {
				if volume.CSI != nil {
					assert.NotEqualf(t, dtcsi.DriverName, volume.CSI.Driver,
						"pod %s: volume %s still uses Dynatrace CSI driver after migration", pod.Name, volume.Name)
				}
			}
		}

		return ctx
	}
}
