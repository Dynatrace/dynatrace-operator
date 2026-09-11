// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package codemodules

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/codemodules"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/nodes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// ImageVolume feature using image volumes deployment
func ImageVolume(t *testing.T) features.Feature {
	if !nodes.IsImageVolumesSupported(t) {
		t.Skip("image volume is not supported")
	}
	builder := features.New("cnfs-codemodules-with-image-volumes")
	secretConfig := tenant.GetSingleTenantSecret(t)
	cnfsSpec := codeModulesCloudNativeSpec(t)
	cloudNativeDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-image-volumes"),
		dynakubeComponents.WithAnnotations(map[string]string{exp.OAImageVolumeKey: "true"}),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cnfsSpec),
	)

	dynakubeComponents.Install(builder, &secretConfig, cloudNativeDynakube)

	sampleApp := sample.NewApp(t, &cloudNativeDynakube, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check injection with image volume of additional pod", codemodules.CheckImageVolumeInjection(sampleApp, cnfsSpec.CodeModulesImage))
	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}
