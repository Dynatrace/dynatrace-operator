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
Create chart name and version as used by the chart label.
*/}}
{{- define "dynatrace-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Check if default image is used
*/}}
{{- define "dynatrace-operator.image" -}}
{{- if .Values.image -}}
	{{- printf "%s" .Values.image -}}
{{- else -}}
	{{- if eq (include "dynatrace-operator.platform" .) "google-marketplace" -}}
    	{{- printf "%s:%s" "gcr.io/dynatrace-marketplace-prod/dynatrace-operator" .Chart.AppVersion }}
	{{- else -}}
		{{- printf "%s:v%s" "docker.io/dynatrace/dynatrace-operator" .Chart.AppVersion }}
	{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Check if we are generating only a part of the yamls
*/}}
{{- define "dynatrace-operator.partial" -}}
	{{- if (default false .Values.partial) -}}
		{{- printf "%s" .Values.partial -}}
	{{- else -}}
	    {{- printf "false" -}}
	{{- end -}}
{{- end -}}
