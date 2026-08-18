# Copyright Dynatrace LLC
# SPDX-License-Identifier: Apache-2.0

{{/*
Check if we need the csi driver.
*/}}
{{- define "dynatrace-operator.needCSI" -}}
	{{- if or (.Values.csidriver.enabled) -}}
		{{- printf "true" -}}
	{{- end -}}
{{- end -}}

{{/*
CSI PriorityClassName
*/}}
{{- define "dynatrace-operator.CSIPriorityClassName" -}}
	{{- default "dynatrace-high-priority" .Values.csidriver.existingPriorityClassName -}}
{{- end -}}

{{/*
Check if we need the csi default priority class
*/}}
{{- define "dynatrace-operator.needPriorityClass" -}}
	{{- if and (include "dynatrace-operator.needCSI" .) (not .Values.csidriver.existingPriorityClassName) -}}
		{{- printf "true" -}}
	{{- end -}}
{{- end -}}

{{/*
CSI plugin-dir path
*/}}
{{- define "dynatrace-operator.CSIPluginDir" -}}
	{{ printf "%s/plugins/csi.oneagent.dynatrace.com/" (trimSuffix "/" (default "/var/lib/kubelet" .Values.csidriver.kubeletPath)) }}
{{- end -}}


{{/*
CSI data-dir path
*/}}
{{- define "dynatrace-operator.CSIDataDir" -}}
	{{ printf "%s/data" (trimSuffix "/" (include "dynatrace-operator.CSIPluginDir" .)) }}
{{- end -}}

{{/*
CSI socket path
*/}}
{{- define "dynatrace-operator.CSISocketPath" -}}
	{{ printf "%s/csi.sock" (trimSuffix "/" (include "dynatrace-operator.CSIPluginDir" .)) }}
{{- end -}}

{{/*
CSI mountpoint-dir path
*/}}
{{- define "dynatrace-operator.CSIMountPointDir" -}}
	{{ printf "%s/pods/" (trimSuffix "/" (default "/var/lib/kubelet" .Values.csidriver.kubeletPath)) }}
{{- end -}}

{{/*
CSI registration-dir path
*/}}
{{- define "dynatrace-operator.CSIRegistrationDir" -}}
	{{ printf "%s/plugins_registry/" (trimSuffix "/" (default "/var/lib/kubelet" .Values.csidriver.kubeletPath)) }}
{{- end -}}
