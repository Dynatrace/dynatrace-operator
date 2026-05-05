package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/stretchr/testify/assert"
)

const (
	expectedBaseInitArgsLen            = 12
	expectedBaseInitArgsLenWithoutMEID = 10
)

func assertResourceAttrArgsAreSorted(t *testing.T, args []string, attrs map[string]string, templateArgs []string, expectedAttrs []string) {
	t.Helper()

	offset := len(templateArgs)
	attrArgs := args[len(args)-len(attrs)-offset : len(args)-offset]
	assert.Equal(t, expectedAttrs, attrArgs)
}

func Test_getInitArgs(t *testing.T) {
	tests := []struct {
		name             string
		meID             string
		clusterName      string
		templateArgs     []string
		resourceAttrs    map[string]string
		expectedLen      int
		mustContain      []string
		expectedAttrArgs []string
	}{
		{
			name:        "base args with MEID",
			meID:        "test-me-id",
			clusterName: "test-cluster-name",
			expectedLen: expectedBaseInitArgsLen,
		},
		{
			name:        "base args without MEID",
			expectedLen: expectedBaseInitArgsLenWithoutMEID,
		},
		{
			name:        "user-defined template args appended",
			meID:        "test-me-id",
			clusterName: "test-cluster-name",
			templateArgs: []string{
				"customArg1",
				"customArg2",
			},
			expectedLen: expectedBaseInitArgsLen + 2,
			mustContain: []string{
				"customArg1",
				"customArg2",
			},
		},
		{
			name: "resourceAttributes propagated as sorted -p args",
			resourceAttrs: map[string]string{
				"team":    "platform",
				"env":     "staging",
				"service": "logmodule",
			},
			expectedLen: expectedBaseInitArgsLenWithoutMEID + 3,
			expectedAttrArgs: []string{
				"-p env=staging",
				"-p service=logmodule",
				"-p team=platform",
			},
		},
		{
			name: "user-defined template args and resourceAttributes combined",
			templateArgs: []string{
				"customArg1",
				"customArg2",
			},
			resourceAttrs: map[string]string{
				"team": "platform",
				"env":  "staging",
			},
			expectedLen: expectedBaseInitArgsLenWithoutMEID + 2 + 2,
			mustContain: []string{
				"customArg1",
				"customArg2",
			},
			expectedAttrArgs: []string{
				"-p env=staging",
				"-p team=platform",
			},
		},
		{
			name:        "resourceAttributes and MEID args combined",
			meID:        "test-me-id",
			clusterName: "test-cluster-name",
			resourceAttrs: map[string]string{
				"service": "logmodule",
				"team":    "platform",
				"env":     "staging",
			},
			expectedLen: expectedBaseInitArgsLen + 3,
			mustContain: []string{
				"-p k8s.cluster.name=$(K8S_CLUSTER_NAME)",
				"-p dt.entity.kubernetes_cluster=$(DT_ENTITY_KUBERNETES_CLUSTER)",
			},
			expectedAttrArgs: []string{
				"-p env=staging",
				"-p service=logmodule",
				"-p team=platform",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dk := dynakube.DynaKube{}
			dk.Name = "dk-name-test"
			dk.Status.KubernetesClusterMEID = tt.meID
			dk.Status.KubernetesClusterName = tt.clusterName
			dk.Spec.ResourceAttributes = tt.resourceAttrs

			if tt.templateArgs != nil {
				dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{Args: tt.templateArgs}
			}

			args := getInitArgs(dk)

			assert.Len(t, args, tt.expectedLen)

			for _, arg := range tt.mustContain {
				assert.Contains(t, args, arg)
			}

			if tt.expectedAttrArgs != nil {
				assertResourceAttrArgsAreSorted(t, args, tt.resourceAttrs, tt.templateArgs, tt.expectedAttrArgs)
			}
		})
	}
}
