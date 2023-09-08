package steps

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	dynakube2 "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type BuilderFunc func(builder *features.FeatureBuilder)
type EnvironmentOptionFunc func() (setupFunc, teardownFunc BuilderFunc)

func CreateNamespaceWithoutTeardown(n v1.Namespace) EnvironmentOptionFunc {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("create operator namespace", namespace.Create(n))
			},
			func(builder *features.FeatureBuilder) {
			}
	}
}
func CreateNamespace(n v1.Namespace) EnvironmentOptionFunc {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("create operator namespace", namespace.Create(n))
			},
			func(builder *features.FeatureBuilder) {
				builder.Teardown(namespace.Delete(n.Name))
			}
	}
}

func DeployOperatorViaMake(namespaceName string, withCSIDriver bool) EnvironmentOptionFunc {
	return func() (_, _ BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("operator manifests installed", operator.InstallViaMake(withCSIDriver))
				builder.Assess("operator started", operator.WaitForDeployment(namespaceName))
				builder.Assess("webhook started", webhook.WaitForDeployment(namespaceName))
				if withCSIDriver {
					builder.Assess("csi driver started", csi.WaitForDaemonset(namespaceName))
				}
			},
			func(builder *features.FeatureBuilder) {
				if withCSIDriver {
					builder.WithTeardown("clean up csi driver files", csi.CleanUpEachPod(namespaceName))
				}

				builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(withCSIDriver))
			}
	}
}

func CreateDynakube(secret tenant.Secret, dk dynakube.DynaKube) EnvironmentOptionFunc {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				assess.CreateDynakube(builder, &secret, dk)
				assess.VerifyDynakubeStartup(builder, dk)
			},
			func(builder *features.FeatureBuilder) {
				builder.WithTeardown("dynakube deleted", dynakube2.Delete(dk))
				if dk.NeedsOneAgent() {
					builder.WithTeardown("oneagent pods stopped", oneagent.WaitForDaemonSetPodsDeletion(dk))
				}
				//teardown.UninstallOperatorFromSource(builder, dk)
			}
	}
}
func CreateFeatureEnvironment(builder *features.FeatureBuilder, opts ...EnvironmentOptionFunc) {
	createSetupSteps(builder, opts)
	createTeardownSteps(builder, opts)
}

func createTeardownSteps(builder *features.FeatureBuilder, opts []EnvironmentOptionFunc) {
	for i := len(opts) - 1; i > 0; i-- {
		_, td := opts[i]()
		td(builder)
	}
}

func createSetupSteps(builder *features.FeatureBuilder, opts []EnvironmentOptionFunc) {
	for _, opt := range opts {
		setup, _ := opt()
		setup(builder)
	}
}
