// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package cloudnative

import (
	"testing"

	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/pgc"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func HostAgentPGC(t *testing.T) features.Feature {
	builder := features.New("host-agent-pgc")
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *componentDynakube.New(
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithActiveGate(),
		componentDynakube.WithCloudNativeSpec(DefaultCloudNativeSpec()),
	)

	componentDynakube.Install(builder, &secretConfig, testDynakube)
	builder.Assess("OneAgent started", k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("PGC file present in OA pods", pgc.WaitForFileInAllPods(testDynakube))

	return builder.Feature()
}
