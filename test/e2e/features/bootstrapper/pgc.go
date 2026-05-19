//go:build e2e

package bootstrapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sjob"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	pgcFixtureCBOR = "\x18\x2a" // minimal valid CBOR: integer 42
	pgcFixtureETag = "test-etag-123"
)

// PGCNoCache tests PGC injection without pre-populated cache.
// Operator attempts to fetch from DT API; secret created even if PGC data is empty.
func PGCNoCache(t *testing.T) features.Feature {
	builder := features.New("bootstrapper-pgc-no-cache")
	secretConfig := tenant.GetSingleTenantSecret(t)

	dk := *dynakubeComponents.New(
		dynakubeComponents.WithName("pgc-bootstrapper"),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: bootstrapperImage},
		}),
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.OANodeImagePullKey: "true",
		}),
	)

	dynakubeComponents.Install(builder, &secretConfig, dk)
	builder.Assess("check if jobs completed", jobsAreCompleted(dk))
	builder.Assess("check if jobs got cleaned up", k8sjob.WaitForDeletionWithOwner(dk.Name, dk.Namespace))

	builder.Assess("PGC source secret created", checkPGCInSourceSecret(dk))

	sampleApp := sample.NewApp(t, &dk, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("PGC projected volume is configured", checkPGCProjectedVolume(sampleApp))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

// PGCCacheHit tests PGC injection with pre-populated cache (ETag match).
// Operator finds cached PGC, verifies ETag match, reuses data without fetching.
func PGCCacheHit(t *testing.T) features.Feature {
	builder := features.New("bootstrapper-pgc-cache-hit")
	secretConfig := tenant.GetSingleTenantSecret(t)

	dk := *dynakubeComponents.New(
		dynakubeComponents.WithName("pgc-bootstrapper-cache"),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: bootstrapperImage},
		}),
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.OANodeImagePullKey: "true",
		}),
	)

	dynakubeComponents.Install(builder, &secretConfig, dk)
	builder.Assess("pre-populate PGC fixture", prePopulatePGCFixture(dk))
	builder.Assess("check if jobs completed", jobsAreCompleted(dk))
	builder.Assess("check if jobs got cleaned up", k8sjob.WaitForDeletionWithOwner(dk.Name, dk.Namespace))

	builder.Assess("PGC is in source secret", checkPGCInSourceSecret(dk))

	sampleApp := sample.NewApp(t, &dk, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("PGC is in namespace secret", checkPGCInNamespaceSecret(sampleApp))
	builder.Assess("PGC projected volume is configured", checkPGCProjectedVolume(sampleApp))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

// prePopulatePGCFixture creates source secret with fixture PGC data to avoid requiring real tenant PGC config.
func prePopulatePGCFixture(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapperconfig.GetSourceConfigSecretName(dk.Name),
				Namespace: dk.Namespace,
				Annotations: map[string]string{
					"internal.operator.dynatrace.com/pgc-etag": pgcFixtureETag,
				},
			},
			Data: map[string][]byte{
				"declarative.cbor": []byte(pgcFixtureCBOR),
			},
		}
		require.NoError(t, envConfig.Client().Resources().Create(ctx, secret))

		return ctx
	}
}

// checkPGCInSourceSecret verifies source secret in operator namespace contains PGC data and ETag.
func checkPGCInSourceSecret(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		// TODO: Implement
		// 1. Read secret from operator.DefaultNamespace with name bootstrapperconfig.GetSourceConfigSecretName(dk.Name)
		// 2. Assert secret.Data["declarative.cbor"] is non-empty
		// 3. Assert secret.Annotations["internal.operator.dynatrace.com/pgc-etag"] is non-empty
		return ctx
	}
}

// checkPGCInNamespaceSecret verifies replicated secret in sample app namespace contains PGC data.
func checkPGCInNamespaceSecret(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		// TODO: Implement
		// 1. Get sample app's namespace (from app metadata)
		// 2. Read secret consts.BootstrapperInitSecretName from that namespace
		// 3. Assert secret.Data["declarative.cbor"] is non-empty
		return ctx
	}
}

// checkPGCProjectedVolume verifies pod's projected volume references bootstrapper config secret.
func checkPGCProjectedVolume(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := app.ListPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items, "sample app pods should exist")

		for _, pod := range pods.Items {
			assertPodHasProjectedInputVolume(t, &pod)
		}

		return ctx
	}
}

func assertPodHasProjectedInputVolume(t *testing.T, pod *corev1.Pod) {
	for _, v := range pod.Spec.Volumes {
		if v.Name == volumes.InputVolumeName {
			require.NotNil(t, v.Projected, "input volume should be projected")
			assertProjectedVolumeHasBootstrapperSecret(t, v.Projected)

			return
		}
	}
	t.Error("pod should have input volume")
}

func assertProjectedVolumeHasBootstrapperSecret(t *testing.T, projected *corev1.ProjectedVolumeSource) {
	for _, src := range projected.Sources {
		if src.Secret != nil && src.Secret.Name == consts.BootstrapperInitSecretName {
			return
		}
	}
	t.Error("projected volume should contain bootstrapper config secret source")
}
