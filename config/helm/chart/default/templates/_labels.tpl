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
Selector labels
*/}}
{{- define "dynatrace-operator.futureSelectorLabels" -}}
app.kubernetes.io/name: dynatrace-operator
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
{{- if eq (include "dynatrace-operator.platform" .) "azure-marketplace" }}
azure-extensions-usage-release-identifier: {{ .Release.Name | quote }}
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
Extensions Controller (EEC) labels
*/}}
{{- define "dynatrace-operator.extensionsControllerLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-extensions-controller
{{- end -}}

{{/*
Extensions OpenTelemetry Collector (OTelC) labels
*/}}
{{- define "dynatrace-operator.extensionsOpenTelemetryCollectorLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-extensions-collector
{{- end -}}

{{/*
LogAgent labels
*/}}
{{- define "dynatrace-operator.logMonitoringLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: logmonitoring
{{- end -}}

{{/*
KSPM labels
*/}}
{{- define "dynatrace-operator.kspmLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: kspm
{{- end -}}
