//go:build e2e

package setup

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type BuilderFunc func(builder *features.FeatureBuilder)
type BuilderStep func() (setupFunc, teardownFunc BuilderFunc)

func (step BuilderStep) AddSetupSetup(builder *features.FeatureBuilder) {
	setupFunc, _ := step()
	setupFunc(builder)
}

type BuilderSteps []BuilderStep

func CreateDefault() BuilderSteps {
	return NewEnvironmentSetup(
		CreateDefaultDynatraceNamespace(),
		DeployOperatorViaMake(true))
}

func CreateNamespaceWithoutTeardown(n corev1.Namespace) BuilderStep {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("create operator namespace", namespace.Create(n))
			},
			func(builder *features.FeatureBuilder) {
			}
	}
}

func CreateNamespace(n corev1.Namespace) BuilderStep {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("create operator namespace", namespace.Create(n))
			},
			func(builder *features.FeatureBuilder) {
				builder.Teardown(namespace.Delete(n.Name))
			}
	}
}

func CreateDefaultDynatraceNamespace() BuilderStep {
	namespaceBuilder := namespace.NewBuilder(dynakube.DefaultNamespace)
	return CreateNamespaceWithoutTeardown(namespaceBuilder.Build())
}

func DeployOperatorViaMake(withCSIDriver bool) BuilderStep {
	return func() (_, _ BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("operator manifests installed", operator.InstallViaMake(withCSIDriver))
				assess.VerifyOperatorDeployment(builder, withCSIDriver)
			},
			func(builder *features.FeatureBuilder) {
				if withCSIDriver {
					builder.WithTeardown("clean up csi driver files", csi.CleanUpEachPod(dynakube.DefaultNamespace))
				}

				builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(withCSIDriver))
			}
	}
}

func DeployOperatorViaHelm(releaseTag string, withCSIDriver bool) BuilderStep {
	return func() (_, _ BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("operator manifests installed", operator.InstallViaHelm(releaseTag, withCSIDriver, "dynatrace"))
				assess.VerifyOperatorDeployment(builder, withCSIDriver)
			},
			func(builder *features.FeatureBuilder) {
				if withCSIDriver {
					builder.WithTeardown("clean up csi driver files", csi.CleanUpEachPod(dynakube.DefaultNamespace))
				}

				builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(withCSIDriver))
			}
	}
}

func CreateDynakube(secret tenant.Secret, dk dynatracev1beta1.DynaKube) BuilderStep {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				assess.CreateDynakube(builder, &secret, dk)
				assess.VerifyDynakubeStartup(builder, dk)
			},
			func(builder *features.FeatureBuilder) {
				builder.WithTeardown("dynakube deleted", dynakube.Delete(dk))
				if dk.NeedsOneAgent() {
					builder.WithTeardown("oneagent pods stopped", oneagent.WaitForDaemonSetPodsDeletion(dk))
				}
				if dk.ClassicFullStackMode() {
					teardown.AddClassicCleanUp(builder, dk)
				}
			}
	}
}

type ManifestInstallationOption func() features.Func

func InstallManifestFromFile(deploymentPath string) BuilderStep {
	return func() (setupFunc, teardownFunc BuilderFunc) {
		return func(builder *features.FeatureBuilder) {
				builder.Assess("installed manifests", manifests.InstallFromFile(deploymentPath))
			},
			func(builder *features.FeatureBuilder) {
				builder.WithTeardown("uninstalled manifests", manifests.UninstallFromFile(deploymentPath))
			}
	}
}

func NewEnvironmentSetup(opts ...BuilderStep) BuilderSteps {
	return opts
}

func CreateFeatureEnvironment(builder *features.FeatureBuilder, opts ...BuilderStep) {
	funcs := BuilderSteps(opts)
	funcs.CreateSetupSteps(builder)
	funcs.CreateTeardownSteps(builder)
}

func (opts BuilderSteps) CreateTeardownSteps(builder *features.FeatureBuilder) {
	for i := len(opts) - 1; i > 0; i-- {
		_, td := opts[i]()
		td(builder)
	}
}

func (opts BuilderSteps) CreateSetupSteps(builder *features.FeatureBuilder) {
	for _, opt := range opts {
		setup, _ := opt()
		setup(builder)
	}
}
