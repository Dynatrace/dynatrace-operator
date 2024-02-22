//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	dtReleaseVersion      = "DT_RELEASE_VERSION"
	dtReleaseProduct      = "DT_RELEASE_PRODUCT"
	dtReleaseStage        = "DT_RELEASE_STAGE"
	dtReleaseBuildVersion = "DT_RELEASE_BUILD_VERSION"

	disabledBuildLabelsNamespace  = "disabled-build-labels-namespace"
	defaultBuildLabelsNamespace   = "default-build-labels-namespace"
	customBuildLabelsNamespace    = "custom-build-labels-namespace"
	preservedBuildLabelsNamespace = "preserved-build-labels-namespace"
	invalidBuildLabelsNamespace   = "invalid-build-labels-namespace"
)

type buildLabel struct {
	reference string
	value     string
}

var (
	namespaceToExpectedLabels = map[string]map[string]buildLabel{
		disabledBuildLabelsNamespace:  disabledBuildLabels,
		defaultBuildLabelsNamespace:   defaultBuildLabels,
		customBuildLabelsNamespace:    customBuildLabels,
		preservedBuildLabelsNamespace: preservedCustomBuildLabels,
		invalidBuildLabelsNamespace:   invalidCustomBuildLabels,
	}

	disabledBuildLabels = map[string]buildLabel{
		dtReleaseVersion:      {"", ""},
		dtReleaseProduct:      {"", ""},
		dtReleaseStage:        {"", ""},
		dtReleaseBuildVersion: {"", ""},
	}

	defaultBuildLabels = map[string]buildLabel{
		dtReleaseVersion:      {"metadata.labels['app.kubernetes.io/version']", "app-kubernetes-io-version"},
		dtReleaseProduct:      {"metadata.labels['app.kubernetes.io/part-of']", "app-kubernetes-io-part-of"},
		dtReleaseStage:        {"", ""},
		dtReleaseBuildVersion: {"", ""},
	}

	customBuildLabels = map[string]buildLabel{
		dtReleaseVersion:      {"metadata.labels['my.domain/version']", "my-domain-version"},
		dtReleaseProduct:      {"metadata.labels['my.domain/product']", "my-domain-product"},
		dtReleaseStage:        {"metadata.labels['my.domain/stage']", "my-domain-stage"},
		dtReleaseBuildVersion: {"metadata.labels['my.domain/build-version']", "my-domain-build-version"},
	}

	preservedCustomBuildLabels = map[string]buildLabel{
		dtReleaseVersion:      {"metadata.labels['my-version']", "my-version"},
		dtReleaseProduct:      {"metadata.labels['my-product']", "my-product"},
		dtReleaseStage:        {"metadata.labels['my-stage']", "my-stage"},
		dtReleaseBuildVersion: {"metadata.labels['my-build-version']", "my-build-version"},
	}

	invalidCustomBuildLabels = map[string]buildLabel{
		dtReleaseVersion: {"metadata.labels['app.kubernetes.io/version']", "app-kubernetes-io-version"},
		dtReleaseProduct: {"metadata.labels['app.kubernetes.io/part-of']", "app-kubernetes-io-part-of"},
		// invalid name of STAGE label -> reference exists but actual label doesn't exist otherwise value would be "my-domain-stage"
		dtReleaseStage: {"metadata.labels['my.domain/invalid-stage']", ""},
		// invalid name of BUILD VERSION label -> reference exists but actual label doesn't exist otherwise value would be "my-domain-build-version"
		dtReleaseBuildVersion: {"metadata.labels['my.domain/invalid-build-version']", ""},
	}
)

// Verification that build labels are created and set accordingly. The test checks:
//
// - default behavior - feature flag exists, but no additional configuration so the default variables are added
// - custom mapping - feature flag exists, with additional configuration so all 4 build variables are added
// - preserved values of existing variables - build variables exist, feature flag exists, with additional configuration, values of build variables not get overwritten
// - incorrect custom mapping - invalid name of BUILD VERSION label, reference exists but actual label doesn't exist
func LabelVersionDetection(t *testing.T) features.Feature {
	builder := features.New("label version")
	builder.WithLabel("name", "app-label-version")

	secretConfig := tenant.GetSingleTenantSecret(t)
	defaultDynakube := *dynakube.New(
		dynakube.WithName("dynakube-default"),
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithNameBasedNamespaceSelector(),
		dynakube.WithApplicationMonitoringSpec(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		}),
	)

	labelVersionDynakube := *dynakube.New(
		dynakube.WithName("dynakube-labels"),
		dynakube.WithAnnotations(map[string]string{dynatracev1beta1.AnnotationFeatureLabelVersionDetection: "true"}),
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithNameBasedNamespaceSelector(),
		dynakube.WithApplicationMonitoringSpec(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		}),
	)

	sampleApps := []*sample.App{
		buildDisabledBuildLabelSampleApp(t, defaultDynakube),
		buildDefaultBuildLabelSampleApp(t, labelVersionDynakube),
		buildCustomBuildLabelSampleApp(t, labelVersionDynakube),
		buildPreservedBuildLabelSampleApp(t, labelVersionDynakube),
		buildInvalidBuildLabelSampleApp(t, labelVersionDynakube),
	}
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, defaultDynakube)
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, labelVersionDynakube)

	// Register actual test (+sample cleanup)
	installSampleApplications(builder, sampleApps)
	checkBuildLabels(builder, sampleApps)
	teardownSampleApplications(builder, sampleApps)

	dynakube.Delete(builder, helpers.LevelTeardown, defaultDynakube)
	dynakube.Delete(builder, helpers.LevelTeardown, labelVersionDynakube)

	return builder.Feature()
}

func installSampleApplications(builder *features.FeatureBuilder, sampleApps []*sample.App) {
	for _, sampleApp := range sampleApps {
		builder.Assess(sampleApp.Name()+" is ready", sampleApp.Install())
	}
}

func teardownSampleApplications(builder *features.FeatureBuilder, sampleApps []*sample.App) {
	for _, sampleApp := range sampleApps {
		builder.WithTeardown(sampleApp.Name()+" is uninstalled", sampleApp.Uninstall())
	}
}

func checkBuildLabels(builder *features.FeatureBuilder, sampleApps []*sample.App) {
	for _, sampleApp := range sampleApps {
		builder.Assess("checking "+sampleApp.Name(), assertBuildLabels(sampleApp, namespaceToExpectedLabels[sampleApp.Namespace()]))
	}
}

func assertBuildLabels(sampleApp *sample.App, expectedBuildLabels map[string]buildLabel) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		kubeResources := envConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, kubeResources)

		for _, podItem := range pods.Items {
			podItem := podItem

			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)

			appContainer := podItem.Spec.Containers[0]
			assert.Equal(t, sampleApp.ContainerName(), appContainer.Name, "%s namespace", sampleApp.Namespace())

			assertReferences(t, &podItem, sampleApp, expectedBuildLabels)

			assertValues(ctx, t, envConfig.Client().Resources(), podItem, sampleApp, expectedBuildLabels)
		}

		return ctx
	}
}

func assertReferences(t *testing.T, pod *corev1.Pod, sampleApp *sample.App, expectedBuildLabels map[string]buildLabel) {
	require.NotNil(t, pod)
	require.NotNil(t, pod.Spec)

	appContainer := pod.Spec.Containers[0]
	require.Equal(t, sampleApp.ContainerName(), appContainer.Name)

	variablesFound := map[string]bool{}

	for _, containerEnvVar := range appContainer.Env {
		if value, hasLabel := expectedBuildLabels[containerEnvVar.Name]; hasLabel {
			if value.reference != "" {
				require.NotNil(t, containerEnvVar.ValueFrom, "%s:%s pod - %s variable has empty ValueFrom property", pod.Namespace, pod.Name, containerEnvVar.Name)
				require.NotNil(t, containerEnvVar.ValueFrom.FieldRef, "%s:%s pod - %s variable has empty FieldRef property", pod.Namespace, pod.Name, containerEnvVar.Name)
				assert.Equal(t, value.reference, containerEnvVar.ValueFrom.FieldRef.FieldPath, "%s:%s pod - %s variable has invalid value reference", pod.Namespace, pod.Name, containerEnvVar.Name)
				variablesFound[containerEnvVar.Name] = true
			}
		}
	}

	for name, value := range expectedBuildLabels {
		_, hasLabel := variablesFound[name]
		if value.reference == "" {
			assert.False(t, hasLabel, "%s:%s pod - %s variable found", pod.Namespace, pod.Name, name)
		} else {
			assert.True(t, hasLabel, "%s:%s pod - %s variable not found", pod.Namespace, pod.Name, name)
		}
	}
}

func assertValues(ctx context.Context, t *testing.T, resource *resources.Resources, podItem corev1.Pod, sampleApp *sample.App, expectedBuildLabels map[string]buildLabel) { //nolint:revive // argument-limit
	for _, variableName := range []string{dtReleaseVersion, dtReleaseProduct, dtReleaseStage, dtReleaseBuildVersion} {
		assertValue(ctx, t, resource, podItem, sampleApp, variableName, expectedBuildLabels[variableName].value)
	}
}

func assertValue(ctx context.Context, t *testing.T, resource *resources.Resources, podItem corev1.Pod, sampleApp *sample.App, variableName string, expectedValue string) { //nolint:revive // argument-limit
	echoCommand := shell.Shell(shell.Echo(fmt.Sprintf("$%s", variableName)))
	executionResult, err := pod.Exec(ctx, resource, podItem, sampleApp.ContainerName(), echoCommand...)
	require.NoError(t, err)

	stdOut := strings.TrimSpace(executionResult.StdOut.String())
	assert.Zero(t, executionResult.StdErr.Len())
	assert.Equal(t, expectedValue, stdOut, "%s:%s pod - %s variable has invalid value", podItem.Namespace, podItem.Name, variableName)
}

func buildDisabledBuildLabelNamespace(testDynakube dynatracev1beta1.DynaKube) corev1.Namespace {
	return *namespace.New(disabledBuildLabelsNamespace, namespace.WithLabels(testDynakube.NamespaceSelector().MatchLabels))
}

func buildDisabledBuildLabelSampleApp(t *testing.T, testDynakube dynatracev1beta1.DynaKube) *sample.App {
	return sample.NewApp(t, &testDynakube, sample.AsDeployment(), sample.WithNamespace(buildDisabledBuildLabelNamespace(testDynakube)))
}

func buildDefaultBuildLabelNamespace(testDynakube dynatracev1beta1.DynaKube) corev1.Namespace {
	return *namespace.New(defaultBuildLabelsNamespace, namespace.WithLabels(testDynakube.NamespaceSelector().MatchLabels))
}

func buildDefaultBuildLabelSampleApp(t *testing.T, testDynakube dynatracev1beta1.DynaKube) *sample.App {
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(buildDefaultBuildLabelNamespace(testDynakube)),
		sample.WithLabels(map[string]string{
			"app.kubernetes.io/version": "app-kubernetes-io-version",
			"app.kubernetes.io/part-of": "app-kubernetes-io-part-of",
			"my.domain/version":         "my-domain-version",
			"my.domain/product":         "my-domain-product",
			"my.domain/stage":           "my-domain-stage",
			"my.domain/build-version":   "my-domain-build-version",
		}),
	)

	return sampleApp
}

func buildCustomBuildLabelNamespace(testDynakube dynatracev1beta1.DynaKube) corev1.Namespace {
	return *namespace.New(
		customBuildLabelsNamespace,
		namespace.WithLabels(testDynakube.NamespaceSelector().MatchLabels),
		namespace.WithAnnotation(map[string]string{
			"mapping.release.dynatrace.com/version":       "metadata.labels['my.domain/version']",
			"mapping.release.dynatrace.com/product":       "metadata.labels['my.domain/product']",
			"mapping.release.dynatrace.com/stage":         "metadata.labels['my.domain/stage']",
			"mapping.release.dynatrace.com/build-version": "metadata.labels['my.domain/build-version']",
		}),
	)
}

func buildCustomBuildLabelSampleApp(t *testing.T, testDynakube dynatracev1beta1.DynaKube) *sample.App {
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(buildCustomBuildLabelNamespace(testDynakube)),
		sample.WithLabels(map[string]string{
			"app.kubernetes.io/version": "app-kubernetes-io-version",
			"app.kubernetes.io/part-of": "app-kubernetes-io-part-of",
			"my.domain/version":         "my-domain-version",
			"my.domain/product":         "my-domain-product",
			"my.domain/stage":           "my-domain-stage",
			"my.domain/build-version":   "my-domain-build-version",
		}),
	)

	return sampleApp
}

func buildPreservedBuildLabelNamespace(testDynakube dynatracev1beta1.DynaKube) corev1.Namespace {
	return *namespace.New(
		preservedBuildLabelsNamespace,
		namespace.WithLabels(testDynakube.NamespaceSelector().MatchLabels),
		namespace.WithAnnotation(map[string]string{
			"mapping.release.dynatrace.com/version":       "metadata.labels['my.domain/version']",
			"mapping.release.dynatrace.com/product":       "metadata.labels['my.domain/product']",
			"mapping.release.dynatrace.com/stage":         "metadata.labels['my.domain/stage']",
			"mapping.release.dynatrace.com/build-version": "metadata.labels['my.domain/build-version']",
		}),
	)
}

func buildPreservedBuildLabelSampleApp(t *testing.T, testDynakube dynatracev1beta1.DynaKube) *sample.App {
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(buildPreservedBuildLabelNamespace(testDynakube)),
		sample.WithLabels(map[string]string{
			"app.kubernetes.io/version": "app-kubernetes-io-version",
			"app.kubernetes.io/part-of": "app-kubernetes-io-part-of",
			"my.domain/version":         "my-domain-version",
			"my.domain/product":         "my-domain-product",
			"my.domain/stage":           "my-domain-stage",
			"my.domain/build-version":   "my-domain-build-version",
			"my-version":                "my-version",
			"my-product":                "my-product",
			"my-stage":                  "my-stage",
			"my-build-version":          "my-build-version",
		}),
		sample.WithEnvs([]corev1.EnvVar{
			{
				Name: dtReleaseVersion,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['my-version']",
					},
				},
			},
			{
				Name: dtReleaseStage,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['my-stage']",
					},
				},
			},
			{
				Name: dtReleaseProduct,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['my-product']",
					},
				},
			},
			{
				Name: dtReleaseBuildVersion,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['my-build-version']",
					},
				},
			},
		}),
	)

	return sampleApp
}

func buildInvalidBuildLabelNamespace(testDynakube dynatracev1beta1.DynaKube) corev1.Namespace {
	return *namespace.New(
		invalidBuildLabelsNamespace,
		namespace.WithLabels(testDynakube.NamespaceSelector().MatchLabels),
		namespace.WithAnnotation(map[string]string{
			"mapping.release.dynatrace.com/stage":         "metadata.labels['my.domain/invalid-stage']",
			"mapping.release.dynatrace.com/build-version": "metadata.labels['my.domain/invalid-build-version']",
		}),
	)
}

func buildInvalidBuildLabelSampleApp(t *testing.T, testDynakube dynatracev1beta1.DynaKube) *sample.App {
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(buildInvalidBuildLabelNamespace(testDynakube)),
		sample.WithLabels(map[string]string{
			"app.kubernetes.io/version": "app-kubernetes-io-version",
			"app.kubernetes.io/part-of": "app-kubernetes-io-part-of",
			"my.domain/version":         "my-domain-version",
			"my.domain/product":         "my-domain-product",
			"my.domain/stage":           "my-domain-stage",
			"my.domain/build-version":   "my-domain-build-version",
		}),
	)

	return sampleApp
}
