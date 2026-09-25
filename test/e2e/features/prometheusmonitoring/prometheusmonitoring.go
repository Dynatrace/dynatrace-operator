// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package prometheusmonitoring

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	pmapi "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	k8sobject "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	defaultTargetAllocatorRepo       = "public.ecr.aws/dynatrace/otel-target-allocator"
	targetAllocatorImageEnvVar       = "E2E_TARGET_ALLOCATOR_IMAGE"
	targetAllocatorImageDigestEnvVar = "E2E_TARGET_ALLOCATOR_IMAGE_DIGEST"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("lifecycle")
	if os.Getenv("OLM") == "true" {
		t.Skip("Skipping Prometheus tests with OLM installation")
	}

	secretConfig := tenant.GetSingleTenantSecret(t)

	dk := dynakube.New(
		dynakube.WithAPIURL(secretConfig.APIURL),
	)

	pm := &pmapi.PrometheusMonitoring{
		Name:      "monitoring",
		Namespace: operator.DefaultNamespace,
		Spec: pmapi.PrometheusMonitoringSpec{
			DynaKubeName: dk.Name,
			TargetAllocator: pmapi.TargetAllocatorSpec{
				Image: registry.GetLatestImageTagURI(t, defaultTargetAllocatorRepo, targetAllocatorImageEnvVar),
			},
			Scraper: pmapi.ScraperSpec{
				Image: dynakube.GetLatestOTelCollectorImageTagURI(t),
			},
			Gateway: pmapi.GatewaySpec{
				Image: dynakube.GetLatestOTelCollectorImageTagURI(t),
			},
		},
	}

	enablePrometheus(builder)

	dynakube.Install(builder, secretConfig, dk)

	builder.Assess("created PrometheusMonitoring", k8sobject.Create(pm))
	builder.Assess("PrometheusMonitoring becomes ready", waitForPhase(pm, status.Running))
	builder.Assess("gateway is ready", k8sobject.Expect(pm.Gateway().GetStatefulSetName(), pm.Namespace, k8sstatefulset.IsRolloutComplete))
	builder.Assess("scraper is ready", k8sobject.Expect(pm.Scraper().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))
	builder.Assess("target allocator is ready", k8sobject.Expect(pm.TargetAllocator().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))

	var previousTargetAllocatorImage, previousScraperImage, previousGatewayImage string

	builder.Assess("updated PrometheusMonitoring", k8sobject.Update(pm.Name, pm.Namespace, func(t *testing.T, pm *pmapi.PrometheusMonitoring) {
		previousTargetAllocatorImage = pm.Status.TargetAllocator.ResolvedImage
		previousScraperImage = pm.Status.Scraper.ResolvedImage
		previousGatewayImage = pm.Status.Gateway.ResolvedImage

		pm.Spec.TargetAllocator.Image = registry.GetLatestImageDigestURI(t, defaultTargetAllocatorRepo, targetAllocatorImageDigestEnvVar)
		pm.Spec.Scraper.Image = dynakube.GetLatestOTelCollectorImageDigestURI(t)
		pm.Spec.Gateway.Image = dynakube.GetLatestOTelCollectorImageDigestURI(t)
	}))
	builder.Assess("PrometheusMonitoring is deploying", waitForPhase(pm, status.Deploying))
	builder.Assess("PrometheusMonitoring becomes ready after update", waitForPhase(pm, status.Running))
	builder.Assess("gateway is ready after update", k8sobject.Expect(pm.Gateway().GetStatefulSetName(), pm.Namespace, k8sstatefulset.IsRolloutComplete))
	builder.Assess("scraper is ready after update", k8sobject.Expect(pm.Scraper().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))
	builder.Assess("target allocator is ready after update", k8sobject.Expect(pm.TargetAllocator().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))

	builder.Assess("resolved images changed", k8sobject.Expect(pm.Name, pm.Namespace, func(pm *pmapi.PrometheusMonitoring) bool {
		return pm.Status.TargetAllocator.ResolvedImage != previousTargetAllocatorImage &&
			pm.Status.Scraper.ResolvedImage != previousScraperImage &&
			pm.Status.Gateway.ResolvedImage != previousGatewayImage
	}))

	builder.Assess("deleted PrometheusMonitoring", k8sobject.Delete(pm))
	builder.Assess("gateway deleted", k8sobject.WaitForDeletion(gatewayStatefulSet(pm)))
	builder.Assess("scraper deleted", k8sobject.WaitForDeletion(scraperDeployment(pm)))
	builder.Assess("target allocator deleted", k8sobject.WaitForDeletion(targetAllocatorDeployment(pm)))

	// Ensure the object gets cleaned up even if a previous step failed
	builder.Teardown(k8sobject.Delete(pm))

	disablePrometheus(builder)

	return builder.Feature()
}

func PublicRegistry(t *testing.T) features.Feature {
	builder := features.New("public-registry")
	if os.Getenv("OLM") == "true" {
		t.Skip("Skipping Prometheus tests with OLM installation")
	}
	builder.Assess("devregistry pull secret exists", k8sobject.Expect(consts.DevRegistryPullSecretName, operator.DefaultNamespace, k8sobject.SecretExists))

	secretConfig := tenant.GetSingleTenantSecret(t)

	dk := dynakube.New(
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCustomPullSecret(consts.DevRegistryPullSecretName),
	)

	pm := &pmapi.PrometheusMonitoring{
		Name:      "monitoring",
		Namespace: operator.DefaultNamespace,
		Spec: pmapi.PrometheusMonitoringSpec{
			DynaKubeName: dk.Name,
		},
	}

	enablePrometheus(builder)

	dynakube.Install(builder, secretConfig, dk)

	builder.Assess("created PrometheusMonitoring", k8sobject.Create(pm))
	builder.Assess("PrometheusMonitoring becomes ready", waitForPhase(pm, status.Running))
	builder.Assess("gateway is ready", k8sobject.Expect(pm.Gateway().GetStatefulSetName(), pm.Namespace, k8sstatefulset.IsRolloutComplete))
	builder.Assess("scraper is ready", k8sobject.Expect(pm.Scraper().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))
	builder.Assess("target allocator is ready", k8sobject.Expect(pm.TargetAllocator().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))

	builder.Teardown(k8sobject.Delete(pm))

	disablePrometheus(builder)

	return builder.Feature()
}

func gatewayStatefulSet(pm *pmapi.PrometheusMonitoring) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{Name: pm.Gateway().GetStatefulSetName(), Namespace: pm.Namespace}
}

func scraperDeployment(pm *pmapi.PrometheusMonitoring) *appsv1.Deployment {
	return &appsv1.Deployment{Name: pm.Scraper().GetDeploymentName(), Namespace: pm.Namespace}
}

func targetAllocatorDeployment(pm *pmapi.PrometheusMonitoring) *appsv1.Deployment {
	return &appsv1.Deployment{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: pm.Namespace}
}

func waitForPhase(pm *pmapi.PrometheusMonitoring, expectedPhase status.DeploymentPhase) features.Func {
	return k8sobject.Eventually(pm.Name, pm.Namespace, func(pm *pmapi.PrometheusMonitoring) bool {
		return pm.Status.Phase == expectedPhase
	})
}

// TODO: remove this when prometheus is no longer experimental
func enablePrometheus(builder *features.FeatureBuilder) {
	builder.WithSetup("enable prometheus feature", helpers.ToFeatureFunc(operator.InstallLocal(false, helm.WithArgs(
		"--set", "experimental.enablePrometheus=true",
		"--set", "prometheus.installCRDs=true",
		// Enable helm to overwrite pre-installed CRDs
		"--take-ownership",
		"--force-conflicts",
	)), true))
}

// TODO: remove this when prometheus is no longer experimental
func disablePrometheus(builder *features.FeatureBuilder) {
	builder.WithTeardown("disable prometheus feature", helpers.ToFeatureFunc(operator.InstallLocal(false, helm.WithArgs(
		"--set", "experimental.enablePrometheus=false",
		"--set", "prometheus.installCRDs=false",
	)), true))
}
