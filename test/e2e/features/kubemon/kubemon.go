// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package kubemon

import (
	"context"
	"os"
	"testing"

	dynakubeapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	agHelper "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	componentOperator "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func FeatureSplitAG(t *testing.T) features.Feature {
	builder := features.New("kubemon-split-ag-mode")

	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Setup(helpers.ToFeatureFunc(componentOperator.InstallLocal(false, componentOperator.EnableKubemonOperand()...), true))

	testDynakube := *componentDynakube.New(
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithActiveGateModules(
			activegate.RoutingCapability.DisplayName,
			activegate.MetricsIngestCapability.DisplayName,
		),
		componentDynakube.WithKubernetesMonitoringRegistration(),
	)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("generic activegate statefulset is ready",
		k8sstatefulset.IsReady(agHelper.GetActiveGateStateFulSetName(&testDynakube), testDynakube.Namespace))

	builder.Assess("kubemon statefulset is ready",
		k8sstatefulset.IsReady(testDynakube.KubernetesMonitoring().GetStatefulSetName(), testDynakube.Namespace))

	builder.Assess("kubemon authtoken secret exists",
		k8ssecret.Exists(testDynakube.KubernetesMonitoring().GetAuthTokenSecretName(), testDynakube.Namespace))

	builder.Assess("kubemon tenant secret exists",
		k8ssecret.Exists(testDynakube.KubernetesMonitoring().GetTenantSecretName(), testDynakube.Namespace))

	builder.Assess("KubernetesMonitoringAvailable condition is True",
		componentDynakube.WaitForCondition(testDynakube, kubemon.KubeMonAvailableConditionType, metav1.ConditionTrue))

	builder.Assess("dynatrace-kubernetes-monitoring-default ClusterRole exists", func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var cr rbacv1.ClusterRole
		require.NoError(t, envConfig.Client().Resources().Get(ctx, "dynatrace-kubernetes-monitoring-default", "", &cr))

		return ctx
	})

	builder.Assess("dynatrace-kubernetes-monitoring-default ClusterRoleBinding exists", func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var crb rbacv1.ClusterRoleBinding
		require.NoError(t, envConfig.Client().Resources().Get(ctx, "dynatrace-kubernetes-monitoring-default", "", &crb))
		assert.Equal(t, "dynatrace-kubernetes-monitoring-default", crb.RoleRef.Name)

		return ctx
	})

	builder.Assess("dynatrace-activegate ClusterRole and ClusterRoleBinding exist on OLM or OpenShift", func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		isOpenshift, err := platform.NewResolver().IsOpenshift()
		require.NoError(t, err)
		if !isOpenshift && os.Getenv("OLM") != "true" {
			t.Skip("dynatrace-activegate ClusterRole is only rendered on OpenShift or OLM installs")
		}

		var cr rbacv1.ClusterRole
		require.NoError(t, envConfig.Client().Resources().Get(ctx, "dynatrace-activegate", "", &cr))

		var crb rbacv1.ClusterRoleBinding
		require.NoError(t, envConfig.Client().Resources().Get(ctx, "dynatrace-activegate", "", &crb))
		assert.Equal(t, "dynatrace-activegate", crb.RoleRef.Name)

		return ctx
	})

	// remove kubemon from dynakube and make sure it was cleaned up properly.
	// Fetch the live cluster state first so fields like CustomPullSecret that were
	// set by Install (on a local copy) are not lost by a full snapshot replacement.
	builder.Assess("dynakube updated - kubemon removed", func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var currentDK dynakubeapi.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, testDynakube.Name, testDynakube.Namespace, &currentDK))
		currentDK.Spec.KubernetesMonitoring = nil
		require.NoError(t, envConfig.Client().Resources().Update(ctx, &currentDK))

		return ctx
	})

	builder.Assess("kubemon statefulset is deleted",
		k8sstatefulset.WaitForAbsence(testDynakube.KubernetesMonitoring().GetStatefulSetName(), testDynakube.Namespace))

	builder.Assess("kubemon authtoken secret is deleted",
		k8ssecret.WaitForAbsence(testDynakube.KubernetesMonitoring().GetAuthTokenSecretName(), testDynakube.Namespace))

	builder.Assess("kubemon tenant secret is deleted",
		k8ssecret.WaitForAbsence(testDynakube.KubernetesMonitoring().GetTenantSecretName(), testDynakube.Namespace))

	builder.Assess("KubernetesMonitoringAvailable condition is absent",
		componentDynakube.WaitForConditionAbsent(testDynakube, kubemon.KubeMonAvailableConditionType))

	builder.Assess("generic activegate statefulset is still ready",
		k8sstatefulset.IsReady(agHelper.GetActiveGateStateFulSetName(&testDynakube), testDynakube.Namespace))

	return builder.Feature()
}

func FeatureRestartTriggers(t *testing.T) features.Feature {
	builder := features.New("kubemon-restart-triggers")

	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Setup(helpers.ToFeatureFunc(componentOperator.InstallLocal(false, componentOperator.EnableKubemonOperand()...), true))

	testDynakube := *componentDynakube.New(
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithKubernetesMonitoringRegistration(),
		componentDynakube.WithUsePublicRegistryFF(),
	)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("kubemon statefulset is ready",
		k8sstatefulset.IsReady(testDynakube.KubernetesMonitoring().GetStatefulSetName(), testDynakube.Namespace))

	builder.Assess("KubernetesMonitoringAvailable condition is True",
		componentDynakube.WaitForCondition(testDynakube, kubemon.KubeMonAvailableConditionType, metav1.ConditionTrue))

	builder.Assess("rotate authtoken secret",
		k8ssecret.Delete(k8ssecret.New(testDynakube.KubernetesMonitoring().GetAuthTokenSecretName(), testDynakube.Namespace, nil)))

	builder.Assess("KubernetesMonitoringAvailable condition is False immediately after secret rotation",
		componentDynakube.WaitForCondition(testDynakube, kubemon.KubeMonAvailableConditionType, metav1.ConditionFalse))

	builder.Assess("KubernetesMonitoringAvailable condition is True after rotation",
		componentDynakube.WaitForCondition(testDynakube, kubemon.KubeMonAvailableConditionType, metav1.ConditionTrue))

	return builder.Feature()
}
