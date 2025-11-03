/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package otlp

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
