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
Check if default image or imageref is used
*/}}
{{- define "dynatrace-operator.image" -}}
{{- if .Values.image -}}
	{{- printf "%s" .Values.image -}}
{{- else -}}
    {{- if (.Values.imageRef).repository -}}
        {{- .Values.imageRef.tag | default (printf "v%s" .Chart.AppVersion) | printf "%s:%s" .Values.imageRef.repository -}}
    {{- else if eq (include "dynatrace-operator.platform" .) "openshift" -}}
        {{- printf "%s:v%s" "registry.connect.redhat.com/dynatrace/dynatrace-operator" .Chart.AppVersion }}
    {{- else if eq (include "dynatrace-operator.platform" .) "google-marketplace" -}}
    	{{- printf "%s:%s" "gcr.io/dynatrace-marketplace-prod/dynatrace-operator" .Chart.AppVersion }}
    {{- else if eq (include "dynatrace-operator.platform" .) "azure-marketplace" -}}
        {{- printf "%s/%s@%s" .Values.global.azure.images.operator.registry .Values.global.azure.images.operator.image .Values.global.azure.images.operator.digest }}
    {{- else if hasPrefix "0.0.0-nightly-" .Chart.AppVersion -}}
        {{- printf "%s:%s" "quay.io/dynatrace/dynatrace-operator" (.Chart.AppVersion | replace "0.0.0-" "") }}
	{{- else -}}
		{{- printf "%s:v%s" "public.ecr.aws/dynatrace/dynatrace-operator" .Chart.AppVersion }}
	{{- end -}}
{{- end -}}
{{- end -}}

{{- define "webhook.securityContext" -}}
    {{- if not .Values.debug -}}
        {{- toYaml .Values.webhook.securityContext -}}
    {{- end -}}
{{- end -}}

{{- define "csidriver.provisioner.resources" -}}
    {{- if not .Values.debug -}}
        {{- toYaml .Values.csidriver.provisioner.resources -}}
    {{- end -}}
{{- end -}}

{{- define "csidriver.server.resources" -}}
    {{- if not .Values.debug -}}
        {{- toYaml .Values.csidriver.server.resources -}}
    {{- end -}}
{{- end -}}

{{- define "dynatrace-operator.startupProbe" -}}
startupProbe:
  exec:
    command:
    - /usr/local/bin/dynatrace-operator
    - startup-probe
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 1
{{- end -}}

{{- define "dynatrace-operator.modules-json-env" -}}
- name: modules.json
  value: |
    {
      "csiDriver": {{ .Values.csidriver.enabled }},
      "activeGate": {{ .Values.rbac.activeGate.create }},
      "oneAgent": {{ .Values.rbac.oneAgent.create }},
      "extensions": {{ .Values.rbac.extensions.create }},
      "logMonitoring": {{ .Values.rbac.logMonitoring.create }},
      "edgeConnect": {{ .Values.rbac.edgeConnect.create }},
      "supportability": {{ .Values.rbac.supportability }},
      "kubernetesMonitoring": {{ .Values.rbac.kubernetesMonitoring.create }},
      "kspm": {{ .Values.rbac.kspm.create }}
    }
{{- end -}}

{{- define "dynatrace-operator.helm-json-env" -}}
- name: helm.json
  value: |
    {
      "tolerations": {{ .Values.csidriver.tolerations | toJson }},
      "annotations": {{ .Values.csidriver.annotations | toJson }},
      "labels": {{ .Values.csidriver.labels | toJson }},
      "job": {
        "securityContext": {{ .Values.csidriver.job.securityContext | toJson }},
        "resources": {{ .Values.csidriver.job.resources | toJson }},
        "priorityClassName": {{ include "dynatrace-operator.CSIPriorityClassName" . | toJson }}
      }
    }
{{- end -}}

{{- define "dynatrace-operator.app-version-env" -}}
- name: APP_VERSION
  value: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{- define "dynatrace-operator.helmPreUpgradeHookAnnotations" -}}
"helm.sh/hook": pre-upgrade
"helm.sh/hook-weight": "-5"
"helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
{{- end -}}
