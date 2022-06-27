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
Selector labels
*/}}
{{- define "dynatrace-operator.futureSelectorLabels" -}}
app.kubernetes.io/name: {{ .Release.Name }}
{{- if not (.Values).manifests }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "dynatrace-operator.commonLabels" -}}
{{ include "dynatrace-operator.futureSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- if not (.Values).manifests }}
helm.sh/chart: {{ include "dynatrace-operator.chart" . }}
{{- end -}}
{{- end -}}

{{/*
Operator labels
*/}}
{{- define "dynatrace-operator.operatorLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: operator
{{- end -}}

{{/*
Operator selector labels
*/}}
{{- define "dynatrace-operator.operatorSelectorLabels" -}}
name: {{ .Release.Name }}
{{- end -}}

{{/*
Webhook labels
*/}}
{{- define "dynatrace-operator.webhookLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: webhook
{{- end -}}

{{/*
Webhook selector labels
*/}}
{{- define "dynatrace-operator.webhookSelectorLabels" -}}
internal.dynatrace.com/component: webhook
internal.dynatrace.com/app: webhook
{{- end -}}

{{/*
CSI labels
*/}}
{{- define "dynatrace-operator.csiLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: csi-driver
{{- end -}}

{{/*
CSI selector labels
*/}}
{{- define "dynatrace-operator.csiSelectorLabels" -}}
internal.oneagent.dynatrace.com/app: csi-driver
internal.oneagent.dynatrace.com/component: csi-driver
{{- end -}}

{{/*
ActiveGate labels
*/}}
{{- define "dynatrace-operator.activegateLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: activegate
{{- end -}}

{{/*
OneAgent labels
*/}}
{{- define "dynatrace-operator.oneagentLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: oneagent
{{- end -}}


{{/*
Check if default image is used
*/}}
{{- define "dynatrace-operator.image" -}}
{{- if .Values.image -}}
	{{- printf "%s" .Values.image -}}
{{- else -}}
	{{- if eq .Values.platform "google" -}}
    	{{- printf "%s:%s" "gcr.io/dynatrace-marketplace-prod/dynatrace-operator" "{{ .Chart.AppVersion }}" }}
	{{- else -}}
		{{- printf "%s:v%s" "docker.io/dynatrace/dynatrace-operator" .Chart.AppVersion }}
	{{- end -}}
{{- end -}}
{{- end -}}


{{/*
Check if we need the csi driver.
*/}}
{{- define "dynatrace-operator.needCSI" -}}
	{{- if or (.Values.csidriver.enabled) (eq (include "dynatrace-operator.partial" .) "csi") -}}
		{{- printf "true" -}}
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
{{- if or (eq .Values.platform "kubernetes") (eq .Values.platform "openshift") (eq .Values.platform "google") (eq .Values.platform "google-autopilot") -}}
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


