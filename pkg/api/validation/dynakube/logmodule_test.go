package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNodeSelector     = "nodeselector"
	testCnfsDynakubeName = "CNFS"
)

func TestConflictingLogModuleConfiguration(t *testing.T) {
	t.Run("oneagent and log module not possible on the same not at the same time in one dk", func(t *testing.T) {
		dk := createLogModuleDynakube(testName, "")
		dk.Spec.OneAgent.CloudNativeFullStack = &dynakube.CloudNativeFullStackSpec{}
		assertDenied(t, []string{errorConflictingOneAgentSpec}, dk, &defaultCSIDaemonSet)
	})
}

func TestConflictingLogModuleNodeSelector(t *testing.T) {
	t.Run("happy cases", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, createLogModuleDynakube(testName, ""))
		assertAllowedWithoutWarnings(t, createLogModuleDynakube(testName, "dd"))

		assertAllowedWithoutWarnings(t, createLogModuleDynakube(testName, "dd"),
			createLogModuleDynakube("other", "othernodeselector"))
		assertAllowedWithoutWarnings(t, createLogModuleDynakube(testName, "dd"),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, "othernodeselector"))
	})

	t.Run("conflict with global oneagent", func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorConflictingLogModule, testCnfsDynakubeName)},
			createLogModuleDynakube(testName, testNodeSelector),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, ""))
	})

	t.Run("conflict with oneagent on the same node", func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorConflictingLogModule, testCnfsDynakubeName)},
			createLogModuleDynakube(testName, testNodeSelector),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, testNodeSelector))
	})

	t.Run("conflict with multiple dynakubes", func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorConflictingLogModule, ""), testCnfsDynakubeName, "conflicting2"},
			createLogModuleDynakube(testName, ""),
			createCloudNativeFullStackDynakube(testCnfsDynakubeName, testNodeSelector),
			createCloudNativeFullStackDynakube("conflicting2", "othernodeselector"))
	})
}

func createLogModuleDynakube(name, nodeSelector string) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			LogModule: dynakube.LogModuleSpec{
				Enabled: true,
			},
		},
	}

	if nodeSelector != "" {
		dk.Spec.Templates.LogModule.NodeSelector = map[string]string{"node": nodeSelector}
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
