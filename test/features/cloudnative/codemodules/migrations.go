//go:build e2e

package codemodules

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func MigrateToImage(t *testing.T) features.Feature {
	builder := features.New("cloudnative-zip-to-image")
	secretConfig := tenant.GetSingleTenantSecret(t)

	appDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("app-codemodules"),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: oneagent.AppInjectionSpec{}}),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
	)

	labels := appDynakube.OneAgent().GetNamespaceSelector().MatchLabels
	sampleNamespace := *namespace.New("codemodules-sample", namespace.WithLabels(labels))

	sampleApp := sample.NewApp(t, &appDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, appDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	appDynakube.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{AppInjectionSpec: *codeModulesAppInjectSpec(t)}

	dynakubeComponents.Update(builder, helpers.LevelAssess, appDynakube)

	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(appDynakube))

	builder.Assess("restart sample app", sampleApp.Restart())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Assess("volumes are mounted correctly", VolumesAreMountedCorrectly(*sampleApp))

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, appDynakube)

	return builder.Feature()
}

func MigrateToNodeImagePull(t *testing.T) features.Feature {
	builder := features.New("cloudnative-zip-to-node-image-pull")
	secretConfig := tenant.GetSingleTenantSecret(t)

	appDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("app-codemodules"),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: oneagent.AppInjectionSpec{}}),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
	)

	labels := appDynakube.OneAgent().GetNamespaceSelector().MatchLabels
	sampleNamespace := *namespace.New("codemodules-sample", namespace.WithLabels(labels))

	sampleApp := sample.NewApp(t, &appDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakubeComponents install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, appDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	appDynakube.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{AppInjectionSpec: *codeModulesAppInjectSpec(t)}
	appDynakube.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}

	dynakubeComponents.Update(builder, helpers.LevelAssess, appDynakube)

	builder.Assess("codemodules have been downloaded", ImageHasBeenDownloaded(appDynakube))

	builder.Assess("restart sample app", sampleApp.Restart())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Assess("volumes are mounted correctly", VolumesAreMountedCorrectly(*sampleApp))

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, appDynakube)

	return builder.Feature()
}
