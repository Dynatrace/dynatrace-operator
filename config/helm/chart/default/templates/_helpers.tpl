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
Common labels
*/}}
{{- define "dynatrace-operator.labels" -}}
helm.sh/chart: {{ include "dynatrace-operator.chart" . }}
dynatrace: operator
operator: dynakube
{{ include "dynatrace-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "dynatrace-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ .Release.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}



{{/*
Check if default image is used
*/}}
{{- define "dynatrace-operator.image" -}}
{{- if .Values.operator.image -}}
	{{- printf "%s" .Values.operator.image -}}
{{- else -}}
	{{- if eq .Values.platform "google" -}}
    	{{- printf "%s:%s" "gcr.io/dynatrace-marketplace-prod/dynatrace-operator" "{{ .Chart.AppVersion }}" }}
	{{- else -}}
		{{- printf "%s:v%s" "docker.io/dynatrace/dynatrace-operator" .Chart.AppVersion }}
	{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Check if only 1 oneagent mode is used.
*/}}
{{- define "dynatrace-operator.modeSet" -}}
	{{- $modes := list .Values.cloudNativeFullStack .Values.classicFullStack .Values.hostMonitoring .Values.applicationMonitoring -}}
	{{- $enabled := dict -}}
		{{- range $index, $mode := $modes -}}
			{{- if $mode -}}
			{{- if $mode.enabled -}}
				{{- $_ := set $enabled ($index|toString) ($mode|toString) -}}
			{{- end -}}
			{{- end -}}
		{{- end -}}
		{{- if (lt (len (keys $enabled)) 2 ) -}}
			{{- "set" -}}
		{{- end -}}
{{- end -}}


{{/*
Check if we need the csi driver.
*/}}
{{- define "dynatrace-operator.needCSI" -}}
	{{- if .Values.cloudNativeFullStack -}}
	  {{- if .Values.cloudNativeFullStack.enabled -}}
		{{- printf "true" -}}
	  {{- end -}}
	{{- end -}}
	{{- if .Values.applicationMonitoring -}}
	  {{- if and .Values.applicationMonitoring.enabled .Values.applicationMonitoring.useCSIDriver -}}
		{{- printf "true" -}}
	  {{- end -}}
	{{- end -}}
{{- end -}}


{{/*
Check if the old and new activeGate sections are used at the same time.
*/}}
{{- define "dynatrace-operator.activeGateModeSet" -}}
    {{- $enabled := dict -}}
	{{- if .Values.activeGate }}
	{{- if .Values.activeGate.capabilities }}
	{{- if ge (len .Values.activeGate.capabilities) 1 }}
		{{- $_ := set $enabled "new" "true" -}}
	{{- end -}}
	{{- end -}}
	{{- end -}}
	{{- if or .Values.kubernetesMonitoring.enabled .Values.routing.enabled }}
		{{- $_ := set $enabled "old" "true" -}}
	{{- end -}}
	{{- if (lt (len (keys $enabled)) 2 ) -}}
			{{- "set" -}}
		{{- end -}}
{{- end -}}


{{/*
Check if platform is set
*/}}
{{- define "dynatrace-operator.platformSet" -}}
{{- if or (eq .Values.platform "kubernetes") (eq .Values.platform "openshift") (eq .Values.platform "google") -}}
    {{ default "set" }}
{{- end -}}
{{- end -}}

{{/*
Common labels webhook
*/}}
{{- define "dynatrace-operator.commonlabelswebhook" -}}
{{ include "dynatrace-operator.selectorLabels" . }}
dynatrace.com/operator: dynakube
internal.dynatrace.com/component: webhook
helm.sh/chart: {{ include "dynatrace-operator.chart" . }}
{{- end -}}
