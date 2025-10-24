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
Set the GOMEMLIMIT envvar to 90% of the configured memory limit. So the go garbage collector is aware, and act accordingly, so we minimize the chance for oomkills.
*/}}
{{- define "dynatrace-operator.gomemlimit" -}}
{{- $limit := (.limits).memory -}}
{{- $number := 0 -}}
{{- $unit := "" -}}
{{- if not $limit -}}
  {{- $number := -1 -}}
{{- else if hasSuffix "Gi" $limit -}}
  {{- $number = int64 (trimSuffix "Gi" $limit) -}}
  {{- $number = mul $number 1024 -}}
{{- else if hasSuffix "G" $limit -}}
  {{- $number = int64 (trimSuffix "G" $limit) -}}
  {{- $number = mul $number 1000 -}}
  {{- $number = int64 (ceil (mulf 0.931323 (float64 $number))) -}}
{{- else if hasSuffix "Mi" $limit -}}
  {{- $number = int64 (trimSuffix "Mi" $limit) -}}
{{- else if hasSuffix "M" $limit -}}
  {{- $number = int64 (trimSuffix "M" $limit) -}}
  {{- $number = int64 (ceil (mulf 0.953674 (float64 $number))) -}}
{{- end -}}
{{- if gt $number 0 -}}
- name: GOMEMLIMIT
  value: {{ printf "%dMiB" (int64 (ceil (mulf 0.9 (float64 $number)))) }}
{{- end -}}
{{- end -}}
