# Copyright Dynatrace LLC
# SPDX-License-Identifier: Apache-2.0

{{/*
Validate if the required related RBACs were enable for kubernetes-monitoring
*/}}
{{- define "validation.rbac.kubemon" -}}
{{- if not .Values.rbac.activeGate.create}}
{{- fail "rbac.activeGate.create = true is required to enable rbac.kubernetesMonitoring.create"}}
{{- end }}
{{- end -}}


{{/*
Validate if the required related RBACs were enabled for kspm
*/}}
{{- define "validation.rbac.kspm" -}}
{{- if not .Values.rbac.kubernetesMonitoring.create}}
{{- fail "rbac.kubernetesMonitoring.create = true is required to enable rbac.kspm.create"}}
{{- end }}
{{- end -}}
