// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package applicationmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/codemodules"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/nodes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// ImageVolumeNoCSI feature using image volumes deployment without CSI driver
func ImageVolumeNoCSI(t *testing.T) features.Feature {
	if !nodes.IsImageVolumesSupported(t) {
		t.Skip("image volume is not supported")
	}

	builder := features.New("app-monitoring-with-image-volumes-without-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)
	codeModuleImage := registry.GetLatestCodeModulesImageTagURI(t)
	appOnlyDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAnnotations(map[string]string{exp.OAImageVolumeKey: "true"}),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
		dynakubeComponents.WithCodeModulesImage(codeModuleImage),
	)

	dynakubeComponents.Install(builder, &secretConfig, appOnlyDynakube)

	sampleApp := sample.NewApp(t, &appOnlyDynakube, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check injection with image volume of additional pod", codemodules.CheckImageVolumeInjection(sampleApp, codeModuleImage))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}
