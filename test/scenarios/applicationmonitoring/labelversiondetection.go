//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	dtReleaseVersion      = "DT_RELEASE_VERSION"
	dtReleaseProduct      = "DT_RELEASE_PRODUCT"
	dtReleaseStage        = "DT_RELEASE_STAGE"
	dtReleaseBuildVersion = "DT_RELEASE_BUILD_VERSION"

	buildLabelsDynakube = "dynakube-labels"

	disabledBuildLabelsNamespace  = "disabled-build-labels-namespace"
	defaultBuildLabelsNamespace   = "default-build-labels-namespace"
	customBuildLablesNamespace    = "custom-build-labels-namespace"
	preservedBuildLablesNamespace = "preserved-build-labels-namespace"
	invalidBuildLabelsNamespace   = "invalid-build-labels-namespace"
)

type buildLabel struct {
	reference string
	value     string
}

var (
	namespaceNames = []string{
		disabledBuildLabelsNamespace,
		defaultBuildLabelsNamespace,
		customBuildLablesNamespace,
		preservedBuildLablesNamespace,
		invalidBuildLabelsNamespace,
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

func installOperator(t *testing.T) features.Feature {
	secretConfig := getSecretConfig(t)

	defaultInstallation := features.New("default installation")

	defaultInstallation.Setup(secrets.ApplyDefault(secretConfig))
	defaultInstallation.Setup(operator.InstallViaMake())
	defaultInstallation.Assess("operator started", operator.WaitForDeployment())
	defaultInstallation.Assess("webhook started", webhook.WaitForDeployment())

	return defaultInstallation.Feature()
}

func installDynakube(t *testing.T, name string, annotations map[string]string) features.Feature {
	secretConfig := getSecretConfig(t)
	defaultInstallation := features.New(name + " installation")

	defaultInstallation.Assess("dynakube applied", dynakube.Apply(dynakube.NewBuilder().
		Name(name).
		Namespace(dynakube.Namespace).
		WithAnnotations(annotations).
		ApiUrl(secretConfig.ApiUrl).
		NamespaceSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				"inject": name,
			},
		}).
		Tokens(dynakube.Name).
		ApplicationMonitoring(&v1beta1.ApplicationMonitoringSpec{}).Build()))
	defaultInstallation.Assess("dynakube phase changes to 'Running'", dynakube.WaitForDynakubePhase(dynakube.NewBuilder().Name(name).Namespace(dynakube.Namespace).Build()))

	return defaultInstallation.Feature()
}

func installSampleApplications() features.Feature {
	defaultInstallation := features.New("sample applications installation")
	defaultInstallation.Assess("sample applications applied", manifests.InstallFromFile("../../testdata/application-monitoring/buildlabels-sample-apps.yaml"))
	for _, namespaceName := range namespaceNames {
		defaultInstallation.Assess(namespaceName+" is ready", deployment.WaitFor(sampleapps.Name, namespaceName))
	}
	return defaultInstallation.Feature()
}

func checkBuildLabels() features.Feature {
	builder := features.New("check build labels")
	builder.Assess("disabled", assertBuildLabels(sampleapps.Namespace, disabledBuildLabels))
	builder.Assess("default", assertBuildLabels(defaultBuildLabelsNamespace, defaultBuildLabels))
	builder.Assess("custom", assertBuildLabels(customBuildLablesNamespace, customBuildLabels))
	builder.Assess("preserved", assertBuildLabels(preservedBuildLablesNamespace, preservedCustomBuildLabels))
	builder.Assess("invalid", assertBuildLabels(invalidBuildLabelsNamespace, invalidCustomBuildLabels))
	return builder.Feature()
}

func getSecretConfig(t *testing.T) secrets.Secret {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())

	require.NoError(t, err)

	return secretConfig
}

func assertBuildLabels(namespaceName string, expectedBuildLabels map[string]buildLabel) func(context.Context, *testing.T, *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := pod.List(t, ctx, resources, namespaceName)

		for _, podItem := range pods.Items {
			podItem := podItem

			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)

			appContainer := podItem.Spec.Containers[0]
			assert.Equal(t, sampleapps.Name, appContainer.Name, "%s namespace", namespaceName)

			assertReferences(t, &podItem, expectedBuildLabels)

			assertValues(t, environmentConfig.Client().RESTConfig(), podItem, expectedBuildLabels)
		}

		return ctx
	}
}

func assertReferences(t *testing.T, pod *corev1.Pod, expectedBuildLabels map[string]buildLabel) {
	require.NotNil(t, pod)
	require.NotNil(t, pod.Spec)

	appContainer := pod.Spec.Containers[0]
	require.Equal(t, "myapp", appContainer.Name)

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

func assertValues(t *testing.T, restConfig *rest.Config, podItem corev1.Pod, expectedBuildLabels map[string]buildLabel) {
	for _, variableName := range []string{dtReleaseVersion, dtReleaseProduct, dtReleaseStage, dtReleaseBuildVersion} {
		assertValue(t, restConfig, podItem, variableName, expectedBuildLabels[variableName].value)
	}
}

func assertValue(t *testing.T, restConfig *rest.Config, podItem corev1.Pod, variableName string, expectedValue string) { //nolint:revive // argument-limit
	executionQuery := pod.NewExecutionQuery(podItem, sampleapps.Name, shell.Shell(shell.Echo(fmt.Sprintf("$%s", variableName)))...)
	executionResult, err := executionQuery.Execute(restConfig)
	require.NoError(t, err)

	stdOut := strings.TrimSpace(executionResult.StdOut.String())
	assert.Zero(t, executionResult.StdErr.Len())
	assert.Equal(t, expectedValue, stdOut, "%s:%s pod - %s variable has invalid value", podItem.Namespace, podItem.Name, variableName)
}
