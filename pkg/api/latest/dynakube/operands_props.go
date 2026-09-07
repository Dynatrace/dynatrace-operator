// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
)

func (dk *DynaKube) ActiveGate() *activegate.ActiveGate {
	// The stored API URL is only used to derive the tenant image registry host, which is only
	// available on 2nd gen URLs. Pass the raw value so a 3rd gen URL yields a non-functional
	// registry host on purpose (see DynaKube.APIURLHost).
	dk.Spec.ActiveGate.SetAPIURL(dk.Spec.APIURL)
	dk.Spec.ActiveGate.SetName(dk.Name)
	dk.Spec.ActiveGate.SetAutomaticTLSCertificate(dk.FF().IsActiveGateAutomaticTLSCertificate())
	dk.Spec.ActiveGate.SetExtensionsDependency(dk.Extensions().IsAnyEnabled())

	return &activegate.ActiveGate{
		Spec:   &dk.Spec.ActiveGate,
		Status: &dk.Status.ActiveGate,
	}
}

func (dk *DynaKube) Extensions() *extensions.Extensions {
	ext := &extensions.Extensions{
		ExecutionController: &dk.Spec.Templates.ExtensionExecutionController,
	}
	if dk.Spec.Extensions != nil {
		ext.Databases = dk.Spec.Extensions.Databases
	}

	// Set required fields for getters that may be called when extensions are disabled.
	ext.SetName(dk.Name)
	ext.SetNamespace(dk.Namespace)

	return ext
}

func (dk *DynaKube) KSPM() *kspm.KSPM {
	_kspm := &kspm.KSPM{
		Spec:                           dk.Spec.KSPM,
		Status:                         &dk.Status.KSPM,
		NodeConfigurationCollectorSpec: &dk.Spec.Templates.KSPMNodeConfigurationCollector,
	}
	_kspm.SetName(dk.GetName())

	return _kspm
}

func (dk *DynaKube) KubernetesMonitoring() *kubemon.KubeMon {
	km := &kubemon.KubeMon{
		Spec:   dk.Spec.KubernetesMonitoring,
		Status: &dk.Status.KubernetesMonitoring,
	}
	km.SetName(dk.Name)
	km.SetAPIURLHost(dk.APIURLHost())

	return km
}

func (dk *DynaKube) IsKubemonEnabled() bool {
	return k8senv.IsKubemonOperandEnabled() && dk.KubernetesMonitoring().IsEnabled()
}

func (dk *DynaKube) IsKubernetesMonitoringEnabled() bool {
	return dk.IsKubemonEnabled() || dk.Spec.ActiveGate.IsKubernetesMonitoringEnabled()
}

func (dk *DynaKube) IsKubernetesMonitoringRegistrationEnabled() bool {
	kubemonRegistrationEnabled := dk.IsKubemonEnabled() && dk.KubernetesMonitoring().IsRegistrationEnabled()
	activeGateRegistrationEnabled := dk.Spec.ActiveGate.IsKubernetesMonitoringEnabled() && dk.FF().IsAutomaticK8sAPIMonitoring()

	return kubemonRegistrationEnabled || activeGateRegistrationEnabled
}

func (dk *DynaKube) LogMonitoring() *logmonitoring.LogMonitoring {
	lm := &logmonitoring.LogMonitoring{
		Spec:         dk.Spec.LogMonitoring,
		TemplateSpec: dk.Spec.Templates.LogMonitoring,
	}
	lm.SetName(dk.Name)
	lm.SetHostAgentDependency(dk.OneAgent().IsDaemonsetRequired())

	return lm
}

func (dk *DynaKube) MetadataEnrichment() *metadataenrichment.MetadataEnrichment {
	return &metadataenrichment.MetadataEnrichment{
		Spec:   &dk.Spec.MetadataEnrichment,
		Status: &dk.Status.MetadataEnrichment,
	}
}

func (dk *DynaKube) OneAgent() *oneagent.OneAgent {
	return oneagent.NewOneAgent(
		&dk.Spec.OneAgent,
		&dk.Status.OneAgent,
		&dk.Status.CodeModules,
		dk.Name,
		dk.APIURLHost(),
		dk.FF().IsOneAgentPrivileged(),
		dk.FF().SkipOneAgentLivenessProbe(),
		dk.GetResourceAttributes(),
	)
}

func (dk *DynaKube) TelemetryIngest() *telemetryingest.TelemetryIngest {
	ts := &telemetryingest.TelemetryIngest{
		Spec: dk.Spec.TelemetryIngest,
	}
	ts.SetName(dk.Name)

	return ts
}

func (dk *DynaKube) OTLPExporterConfiguration() *otlp.ExporterConfiguration {
	return otlp.NewExporterConfiguration(dk.Spec.OTLPExporterConfiguration, dk.Spec.ResourceAttributes)
}
