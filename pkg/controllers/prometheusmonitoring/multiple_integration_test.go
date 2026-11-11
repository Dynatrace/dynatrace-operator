// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/gateway"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/scraper"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/targetallocator"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const testNamespace = "dynatrace"

// TestMultiplePrometheusMonitorings reconciles two PrometheusMonitorings of the same DynaKube in the same namespace
// against a real API server and verifies that their resources are isolated from each other:
// distinct resource names, Service selectors matching only the own pods, and each target allocator being
// configured with its own custom resource selector and scraper pool.
func TestMultiplePrometheusMonitorings(t *testing.T) {
	const (
		dynaKubeName = "dk"
		image        = "registry.example.com/component:1.2.3"
	)

	clt := integrationtests.SetupTestEnvironment(t)
	integrationtests.CreateNamespace(t, clt, testNamespace)

	dk := &dynakube.DynaKube{
		Name: dynaKubeName, Namespace: testNamespace,
		Spec: dynakube.DynaKubeSpec{CustomPullSecret: "custom-pull-secret"},
	}

	newPM := func(name string) *prometheusmonitoring.PrometheusMonitoring {
		pm := &prometheusmonitoring.PrometheusMonitoring{
			Name: name, Namespace: testNamespace,
			Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: dynaKubeName},
		}
		pm.Spec.Gateway.Image = image
		pm.Spec.Scraper.Image = image
		pm.Spec.TargetAllocator.Image = image
		pm.Spec.TargetAllocator.CustomResourceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"pm": name},
		}

		integrationtests.CreateKubernetesObject(t, clt, pm)

		return pm
	}

	pms := []*prometheusmonitoring.PrometheusMonitoring{newPM("pm-one"), newPM("pm-two")}

	for _, pm := range pms {
		require.NoError(t, (&gateway.Reconciler{Client: clt}).Reconcile(t.Context(), pm, dk, nil))
		require.NoError(t, (&targetallocator.Reconciler{Client: clt}).Reconcile(t.Context(), pm, dk, nil))
		require.NoError(t, (&scraper.Reconciler{Client: clt}).Reconcile(t.Context(), pm, dk, nil))
	}

	t.Run("resource names are distinct", func(t *testing.T) {
		assertDistinctResourceNames(t, pms)
	})

	t.Run("services select only their own pods", func(t *testing.T) {
		assertServicesSelectOwnPods(t, clt, pms)
	})

	t.Run("target allocators are configured with their own selectors", func(t *testing.T) {
		assertAllocatorSelectors(t, clt, pms)
	})
}

func assertDistinctResourceNames(t *testing.T, pms []*prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()

	seen := map[string]string{}

	for _, pm := range pms {
		for kind, name := range map[string]string{
			"gateway":         pm.Gateway().GetStatefulSetName(),
			"scraper":         pm.Scraper().GetDeploymentName(),
			"targetallocator": pm.TargetAllocator().GetDeploymentName(),
		} {
			key := kind + "/" + name
			assert.NotContains(t, seen, key)
			seen[key] = pm.Name
		}
	}
}

// collectPodLabels returns the pod template labels of every workload, grouped by owning PrometheusMonitoring.
func collectPodLabels(t *testing.T, clt client.Client, pms []*prometheusmonitoring.PrometheusMonitoring) map[string][]map[string]string {
	t.Helper()

	podLabels := map[string][]map[string]string{}

	for _, pm := range pms {
		sts := &appsv1.StatefulSet{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.Gateway().GetStatefulSetName(), Namespace: testNamespace}, sts))

		scraperDeploy := &appsv1.Deployment{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.Scraper().GetDeploymentName(), Namespace: testNamespace}, scraperDeploy))

		allocatorDeploy := &appsv1.Deployment{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: testNamespace}, allocatorDeploy))

		podLabels[pm.Name] = []map[string]string{sts.Spec.Template.Labels, scraperDeploy.Spec.Template.Labels, allocatorDeploy.Spec.Template.Labels}
	}

	return podLabels
}

func assertServicesSelectOwnPods(t *testing.T, clt client.Client, pms []*prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()

	podLabels := collectPodLabels(t, clt, pms)

	services := map[string]*prometheusmonitoring.PrometheusMonitoring{}

	for _, pm := range pms {
		services[pm.Gateway().GetStatefulSetName()] = pm
		services[pm.TargetAllocator().GetDeploymentName()] = pm
	}

	for name, owner := range services {
		svc := &corev1.Service{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: testNamespace}, svc))
		require.NotEmpty(t, svc.Spec.Selector)

		selector := labels.SelectorFromSet(svc.Spec.Selector)

		matched := 0

		for pmName, pods := range podLabels {
			for _, l := range pods {
				if !selector.Matches(labels.Set(l)) {
					continue
				}

				assert.Equal(t, owner.Name, pmName, "service %s selects a pod of another PrometheusMonitoring", name)

				matched++
			}
		}

		assert.Equal(t, 1, matched, "service %s must select exactly one workload", name)
	}
}

func assertAllocatorSelectors(t *testing.T, clt client.Client, pms []*prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()

	for _, pm := range pms {
		cm := &corev1.ConfigMap{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: testNamespace}, cm))

		var cfg targetallocator.Config
		require.NoError(t, yaml.Unmarshal([]byte(cm.Data["targetallocator.yaml"]), &cfg))

		assert.Equal(t, pm.Spec.TargetAllocator.CustomResourceSelector, cfg.PrometheusCR.PodMonitorSelector)
		assert.Equal(t, map[string]string{"pm": pm.Name}, cfg.PrometheusCR.PodMonitorSelector.MatchLabels)

		// The collector selector must match the pods of the own scraper only.
		scraperDeploy := &appsv1.Deployment{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.Scraper().GetDeploymentName(), Namespace: testNamespace}, scraperDeploy))

		collectorSelector, err := metav1.LabelSelectorAsSelector(cfg.CollectorSelector)
		require.NoError(t, err)
		assert.True(t, collectorSelector.Matches(labels.Set(scraperDeploy.Spec.Template.Labels)))

		for _, other := range pms {
			if other.Name == pm.Name {
				continue
			}

			otherScraper := &appsv1.Deployment{}
			require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: other.Scraper().GetDeploymentName(), Namespace: testNamespace}, otherScraper))
			assert.False(t, collectorSelector.Matches(labels.Set(otherScraper.Spec.Template.Labels)),
				"allocator of %s selects the scraper of %s", pm.Name, other.Name)
		}
	}
}
