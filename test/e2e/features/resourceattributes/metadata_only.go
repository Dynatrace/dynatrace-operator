// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package resourceattributes

import (
	"testing"

	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func MetadataOnly(t *testing.T) features.Feature {
	builder := features.New("resource-attributes-metadata-only")
	secretConfig := tenant.GetSingleTenantSecret(t)
	ns := "resource-attributes-metadata-only"

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithResourceAttributes(globalAttrs),
	)

	sampleApp := newSampleApp(t, &testDynakube, ns, testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	installSampleApp(builder, sampleApp)

	builder.Assess("initcontainer contains args with additionalAttributes", assessInitContainerArgs(sampleApp, globalAttrs))
	builder.Assess("dt_metadata.json and dt_metadata.properties contains merged global resource attributes", assessDTMetadataFiles(testDynakube, sampleApp, globalAttrs))
	builder.Assess("metadata.dynatrace.com JSON annotation contains global resource attributes and workload info", assessPodMetadataJSONAnnotation(sampleApp, globalAttrs))

	uninstallSampleApp(builder, sampleApp)

	return builder.Feature()
}
