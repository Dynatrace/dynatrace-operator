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

package otlpexporterconfiguration

func (s *Spec) IsOverrideEnabled() bool {
	return s.OverrideEnvVars != nil && *s.OverrideEnvVars
}

func (s *Spec) IsMetricsEnabled() bool {
	return s.Signals.Metrics != nil
}

func (s *Spec) IsTracesEnabled() bool {
	return s.Signals.Traces != nil
}

func (s *Spec) IsLogsEnabled() bool {
	return s.Signals.Logs != nil
}
