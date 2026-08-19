# Copyright Dynatrace LLC
# SPDX-License-Identifier: Apache-2.0

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
CRD cleanup labels
*/}}
{{- define "dynatrace-operator.crdStorageMigrationLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: crd-storage-migration
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
{{- define "dynatrace-operator.extensionControllerLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-extension-controller
{{- end -}}

{{/*
OpenTelemetry Collector (OTelC) labels
*/}}
{{- define "dynatrace-operator.openTelemetryCollectorLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-otel-collector
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

{{/*
Database Extensions Executor labels
*/}}
{{- define "dynatrace-operator.databaseDatasourceLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-sql-extension-executor
{{- end -}}

{{/*
Prometheus Target Allocator labels
*/}}
{{- define "dynatrace-operator.targetAllocatorLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-target-allocator
{{- end -}}

{{/*
Prometheus Scraper labels
*/}}
{{- define "dynatrace-operator.scraperLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-prometheus-scraper
{{- end -}}

{{/*
Prometheus Gateway labels
*/}}
{{- define "dynatrace-operator.gatewayLabels" -}}
{{ include "dynatrace-operator.commonLabels" . }}
app.kubernetes.io/component: dynatrace-prometheus-gateway
{{- end -}}
