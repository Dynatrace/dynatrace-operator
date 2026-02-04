// Copyright 2020 Dynatrace LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

{{/*
Little helper to migrate away from .Values.webhook.highAvailability
*/}}
{{- define "dynatrace-operator.webhook.replicas" -}}
	{{- if or (not .Values.webhook.highAvailability) .Values.debug -}}
		{{- 1 -}}
	{{- else -}}
		{{- .Values.webhook.replicas -}}
	{{- end -}}
{{- end -}}


{{- define "dynatrace-operator.webhook.topologySpreadConstraints" -}}
  {{- if and (ge (int (include "dynatrace-operator.webhook.replicas" .)) 2 ) (not (empty .Values.webhook.topologySpreadConstraints)) -}}
topologySpreadConstraints:
  {{- range $constraint := .Values.webhook.topologySpreadConstraints }}
- {{ toYaml $constraint | nindent 2 }}
  labelSelector:
    matchLabels:
      {{- include "dynatrace-operator.webhookSelectorLabels" . | nindent 6 }}
    {{- end }}
  {{- end }}
{{- end }}

{{- define "dynatrace-operator.webhook.podDisruptionBudget" -}}
{{- if and (.Values.webhook.highAvailability) (not (empty .Values.webhook.podDisruptionBudget)) -}}
{{- toYaml .Values.webhook.podDisruptionBudget -}}
{{- end -}}
{{- end -}}
