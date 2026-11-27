// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package hostmonitoring

import (
	"context"
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const k8sClusterEntityField = "dt.entity.kubernetes_cluster"

var expectedMetadataFields = []string{
	"k8s.cluster.name",
	"k8s.cluster.uid",
	"k8s.node.name",
}

func GenerateMetadata(t *testing.T) features.Feature {
	builder := features.New("generate-metadata")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithActiveGate(),
		componentDynakube.WithHostMonitoringSpec(&oneagent.HostInjectSpec{}),
	}
	testDynakube := componentDynakube.New(options...)

	// Register Dynakube install
	componentDynakube.Install(builder, secretConfig, testDynakube)
	builder.Assess("OneAgent started", k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("active gate pod is running", activegate.CheckContainer(testDynakube))

	builder.Assess("Checking if all OneAgent pods have generated metadata", oneAgentHaveGeneratedMetadata(testDynakube))

	return builder.Feature()
}

func oneAgentHaveGeneratedMetadata(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		r := envConfig.Client().Resources()

		require.NoError(t, r.Get(ctx, dk.Name, dk.Namespace, dk))

		q := k8sdaemonset.NewQuery(ctx, r, client.ObjectKey{
			Name:      dk.OneAgent().GetDaemonsetName(),
			Namespace: dk.Namespace,
		})

		err := q.ForEachPod(assertGeneratedMetadataFields(ctx, t, r, dk.HasPlatformToken()))
		require.NoError(t, err)

		return ctx
	}
}

func assertGeneratedMetadataFields(ctx context.Context, t *testing.T, resource *resources.Resources, hasPlatformToken bool) k8sdaemonset.PodConsumer {
	expectedFields := expectedMetadataFields
	if !hasPlatformToken {
		expectedFields = append(slices.Clone(expectedMetadataFields), k8sClusterEntityField)
	}

	return func(pod *corev1.Pod) {
		generatedMetadata := metadataenrichment.GetNodeMetadataPropertiesFromPod(ctx, t, resource, pod)
		assert.NotEmpty(t, generatedMetadata, "generated metadata should not be empty")

		for _, attribute := range expectedFields {
			assert.Containsf(t, generatedMetadata, attribute, "generated metadata should contain %s attribute", attribute)
			assert.NotEmptyf(t, generatedMetadata[attribute], "generated metadata %s attribute should not be empty", attribute)
		}

		if hasPlatformToken {
			assert.NotContainsf(t, generatedMetadata, k8sClusterEntityField, "generated metadata should not contain %s attribute when a platform token is used", k8sClusterEntityField)
		}
	}
}
