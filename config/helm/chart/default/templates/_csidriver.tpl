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
	{{- if and (eq (include "dynatrace-operator.needCSI" .) "true") (not .Values.csidriver.existingPriorityClassName) -}}
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
