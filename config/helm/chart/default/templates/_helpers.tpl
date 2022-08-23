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
	{{- if eq .Values.platform "google-marketplace" -}}
    	{{- printf "%s:%s" "gcr.io/dynatrace-marketplace-prod/dynatrace-operator" "{{ .Chart.AppVersion }}" }}
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


{{/*
Check if platform is set
*/}}
{{- define "dynatrace-operator.platformSet" -}}
{{- if or (eq .Values.platform "kubernetes") (eq .Values.platform "openshift") (eq .Values.platform "google-marketplace") (eq .Values.platform "gke-autopilot") -}}
    {{ default "set" }}
{{- end -}}
{{- end -}}

{{/*
Exclude Kubernetes manifest not running on OLM
*/}}
{{- define "dynatrace-operator.openshiftOrOlm" -}}
{{- if and (or (eq .Values.platform "openshift") (.Values.olm)) (eq (include "dynatrace-operator.partial" .) "false") -}}
    {{ default "true" }}
{{- end -}}
{{- end -}}

{{/*
Check if the platform is set
*/}}
{{- define "dynatrace-operator.platformRequired" -}}
{{- $platformIsSet := printf "%s" (required "Platform needs to be set to kubernetes, openshift, google-marketplace, or gke-autopilot" (include "dynatrace-operator.platformSet" .))}}
{{- end -}}
