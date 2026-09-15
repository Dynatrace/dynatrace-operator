// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package prometheusmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	k8sobject "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	secretConfig := tenant.GetSingleTenantSecret(t)

	dk := *dynakube.New(
		dynakube.WithAPIURL(secretConfig.APIURL),
	)

	pm := &prometheusmonitoring.PrometheusMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: operator.DefaultNamespace,
		},
		Spec: prometheusmonitoring.PrometheusMonitoringSpec{
			DynaKubeRef: dk.Name,
			TargetAllocator: prometheusmonitoring.TargetAllocatorSpec{
				PodSpec: prometheusmonitoring.PodSpec{
					Image: registry.GetLatestImageTagURI(t, defaultTargetAllocatorRepo, targetAllocatorImageEnvVar),
				},
			},
			Scraper: prometheusmonitoring.ScraperSpec{
				PodSpec: prometheusmonitoring.PodSpec{
					Image: dynakube.GetLatestOTelCollectorImageTagURI(t),
				},
			},
			Gateway: prometheusmonitoring.GatewaySpec{
				PodSpec: prometheusmonitoring.PodSpec{
					Image: dynakube.GetLatestOTelCollectorImageTagURI(t),
				},
			},
		},
	}

	enablePrometheus(builder)

	dynakube.Install(builder, &secretConfig, dk)

	builder.Assess("created PrometheusMonitoring", k8sobject.Create(pm))
	builder.Assess("PrometheusMonitoring becomes ready", waitForPhase(pm, status.Running))
	builder.Assess("gateway is ready", k8sobject.Expect(pm.Gateway().GetStatefulSetName(), pm.Namespace, k8sstatefulset.IsRolloutComplete))
	builder.Assess("scraper is ready", k8sobject.Expect(pm.Scraper().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))
	builder.Assess("target allocator is ready", k8sobject.Expect(pm.TargetAllocator().GetDeploymentName(), pm.Namespace, k8sdeployment.IsRolloutComplete))

	builder.Assess("deleted PrometheusMonitoring", k8sobject.Delete(pm))
	builder.Assess("gateway deleted", k8sobject.WaitForDeletion(gatewayStatefulSet(pm)))
	builder.Assess("scraper deleted", k8sobject.WaitForDeletion(scraperDeployment(pm)))
	builder.Assess("target allocator deleted", k8sobject.WaitForDeletion(targetAllocatorDeployment(pm)))

	// Ensure the object get cleaned up even if a previous step failed
	builder.Teardown(k8sobject.Delete(pm))

	disablePrometheus(builder)

	return builder.Feature()
}

func gatewayStatefulSet(pm *prometheusmonitoring.PrometheusMonitoring) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: pm.Gateway().GetStatefulSetName(), Namespace: pm.Namespace}}
}

func scraperDeployment(pm *prometheusmonitoring.PrometheusMonitoring) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: pm.Scraper().GetDeploymentName(), Namespace: pm.Namespace}}
}

func targetAllocatorDeployment(pm *prometheusmonitoring.PrometheusMonitoring) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: pm.Namespace}}
}

func waitForPhase(pm *prometheusmonitoring.PrometheusMonitoring, expectedPhase status.DeploymentPhase) features.Func {
	return k8sobject.Eventually(pm.Name, pm.Namespace, func(pm *prometheusmonitoring.PrometheusMonitoring) bool {
		return pm.Status.Phase == expectedPhase
	})
}

// TODO: remove this when prometheus is no longer experimental
func enablePrometheus(builder *features.FeatureBuilder) {
	builder.WithSetup("enable prometheus feature", helpers.ToFeatureFunc(operator.InstallLocal(false, helm.WithArgs(
		"--set", "experimental.enablePrometheus=true",
		"--set", "prometheus.installCRDs=true",
	)), true))
}

// TODO: remove this when prometheus is no longer experimental
func disablePrometheus(builder *features.FeatureBuilder) {
	builder.WithTeardown("disable prometheus feature", helpers.ToFeatureFunc(operator.InstallLocal(false, helm.WithArgs(
		"--set", "experimental.enablePrometheus=false",
		"--set", "prometheus.installCRDs=false",
	)), true))
}
