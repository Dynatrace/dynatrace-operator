// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otlp

import "github.com/Dynatrace/dynatrace-operator/pkg/api/shared/resourceattributes"

func NewExporterConfiguration(spec *ExporterConfigurationSpec, globalResourceAttributes map[string]string) *ExporterConfiguration {
	return &ExporterConfiguration{
		Spec:                     spec,
		globalResourceAttributes: globalResourceAttributes,
	}
}

func (e *ExporterConfiguration) IsEnabled() bool {
	if e.Spec == nil {
		return false
	}

	return e.IsTracesEnabled() || e.IsMetricsEnabled() || e.IsLogsEnabled()
}

func (e *ExporterConfiguration) IsOverrideEnvVarsEnabled() bool {
	return e.Spec != nil && e.Spec.OverrideEnvVars != nil && *e.Spec.OverrideEnvVars
}

func (e *ExporterConfiguration) IsMetricsEnabled() bool {
	return e.Spec != nil && e.Spec.Signals.Metrics != nil
}

func (e *ExporterConfiguration) IsTracesEnabled() bool {
	return e.Spec != nil && e.Spec.Signals.Traces != nil
}

func (e *ExporterConfiguration) IsLogsEnabled() bool {
	return e.Spec != nil && e.Spec.Signals.Logs != nil
}

func (e *ExporterConfiguration) GetResourceAttributes() map[string]string {
	if e.Spec == nil {
		return e.globalResourceAttributes
	}

	return resourceattributes.Merge(e.globalResourceAttributes, e.Spec.AdditionalResourceAttributes)
}

func (e *ExporterConfiguration) HasAdditionalResourceAttributes() bool {
	return e.Spec != nil && len(e.Spec.AdditionalResourceAttributes) > 0
}

func (e *ExporterConfiguration) GetAdditionalResourceAttributes() map[string]string {
	if e.Spec == nil {
		return nil
	}

	return e.Spec.AdditionalResourceAttributes
}
