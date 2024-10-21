package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNodeSelector     = "nodeselector"
	testCnfsDynakubeName = "CNFS"
)

func TestConflictingLogMonitoringConfiguration(t *testing.T) {
	t.Run("oneagent and log module not possible on the same not at the same time in one dk", func(t *testing.T) {
		dk := createLogMonitoringDynakube(testName, "")
		dk.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{}
		assertDenied(t, []string{errorConflictingOneAgentSpec}, dk, &defaultCSIDaemonSet)
	})
}

func TestConflictingLogMonitoringNodeSelector(t *testing.T) {
	t.Run("happy cases", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, createLogMonitoringDynakube(testName, ""))
		assertAllowedWithoutWarnings(t, createLogMonitoringDynakube(testName, "dd"))

		assertAllowedWithoutWarnings(t, createLogMonitoringDynakube(testName, "dd"),
			createLogMonitoringDynakube("other", "othernodeselector"))
		assertAllowedWithoutWarnings(t, createLogMonitoringDynakube(testName, "dd"),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, "othernodeselector"))
	})

	t.Run("conflict with global oneagent", func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorConflictingLogMonitoring, testCnfsDynakubeName)},
			createLogMonitoringDynakube(testName, testNodeSelector),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, ""))
	})

	t.Run("conflict with oneagent on the same node", func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorConflictingLogMonitoring, testCnfsDynakubeName)},
			createLogMonitoringDynakube(testName, testNodeSelector),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, testNodeSelector))
	})

	t.Run("conflict with multiple dynakubes", func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorConflictingLogMonitoring, ""), testCnfsDynakubeName, "conflicting2"},
			createLogMonitoringDynakube(testName, ""),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, testNodeSelector),
			createCloudNativeFullStackDynakube("conflicting2", "othernodeselector"))
	})
}

func createLogMonitoringDynakube(name, nodeSelector string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:        testApiUrl,
			LogMonitoring: &logmonitoring.Spec{},
		},
	}

	if nodeSelector != "" {
		dk.Spec.Templates.LogMonitoring.NodeSelector = map[string]string{"node": nodeSelector}
	}

	return dk
}

func createCloudNativeFullStackDynakube(name string, nodeSelector string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
		},
	}

	if nodeSelector != "" {
		dk.Spec.OneAgent.CloudNativeFullStack.NodeSelector = map[string]string{"node": nodeSelector}
	}

	return dk
}
