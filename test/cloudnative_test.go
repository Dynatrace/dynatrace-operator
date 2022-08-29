//go:build e2e
// +build e2e

package test

import (
	"context"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/environment"
	"github.com/Dynatrace/dynatrace-operator/test/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

const (
	dynatraceNamespace = "dynatrace"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Setup(deleteDynakubeIfExists())
	testEnvironment.Setup(oneagent.DeleteDaemonsetIfExists())
	testEnvironment.Setup(namespace.Recreate(dynatraceNamespace))

	//testEnvironment.Finish(namespace.Delete(dynatraceNamespace))
	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, install(t))
}

func install(t *testing.T) features.Feature {
	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)

	secretPath := path.Join(currentWorkingDirectory, "/testdata/secrets/cloudnative-install.yaml")
	secretConfig, err := secrets.NewFromConfig(afero.NewOsFs(), secretPath)

	require.NoError(t, err)

	defaultInstallation := features.New("default installation")

	defaultInstallation.Setup(secrets.ApplyDefault(secretConfig))
	defaultInstallation.Setup(operator.InstallForKubernetes)

	defaultInstallation.Assess("operator started", operator.WaitForDeployment())
	defaultInstallation.Assess("webhook started", webhook.WaitForDeployment())
	defaultInstallation.Assess("csi driver started", csi.WaitForDaemonset())
	defaultInstallation.Assess("dynakube applied", applyDynakube(secretConfig))
	defaultInstallation.Assess("activegate started", activegate.WaitForStatefulSet())
	defaultInstallation.Assess("oneagent started", oneagent.WaitForDaemonset())
	defaultInstallation.Assess("dynakube phase changes to 'Running'", waitForDynakubePhase())

	return defaultInstallation.Feature()
}

func dynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dynakube",
			Namespace: "dynatrace",
		},
	}
}

func applyDynakube(secretConfig secrets.Secrets) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))

		instance := dynakube()
		instance.Spec = dynatracev1beta1.DynaKubeSpec{
			APIURL: secretConfig.ApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.DynatraceApiCapability.DisplayName,
					dynatracev1beta1.RoutingCapability.DisplayName,
					dynatracev1beta1.MetricsIngestCapability.DisplayName,
					dynatracev1beta1.StatsdIngestCapability.DisplayName,
				},
			},
		}

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &instance))

		return ctx
	}
}

func deleteDynakubeIfExists() env.Func {
	return func(ctx context.Context, environmentConfig *envconf.Config) (context.Context, error) {
		instance := dynakube()
		resources := environmentConfig.Client().Resources()

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		err = resources.Delete(ctx, &instance)

		if err != nil && !k8serrors.IsNotFound(err) {
			return ctx, errors.WithStack(err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&instance))

		return ctx, err
	}
}

func waitForDynakubePhase() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		instance := dynakube()
		resources := environmentConfig.Client().Resources()

		require.NoError(t, wait.For(conditions.New(resources).ResourceMatch(&instance, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynatracev1beta1.DynaKube)
			return isDynakube && dynakube.Status.Phase == dynatracev1beta1.Running
		})))

		return ctx
	}
}
