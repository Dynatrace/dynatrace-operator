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
	{{- else -}}
		{{- printf "%s:v%s" "public.ecr.aws/dynatrace/dynatrace-operator" .Chart.AppVersion }}
	{{- end -}}
{{- end -}}
{{- end -}}

{{- define "webhook.securityContext" -}}
    {{- if ne .Values.debug true -}}
        {{- toYaml .Values.webhook.securityContext -}}
    {{- end -}}
{{- end -}}

{{- define "csidriver.provisioner.resources" -}}
    {{- if ne .Values.debug true -}}
        {{- toYaml .Values.csidriver.provisioner.resources -}}
    {{- end -}}
{{- end -}}

{{- define "csidriver.server.resources" -}}
    {{- if ne .Values.debug true -}}
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
