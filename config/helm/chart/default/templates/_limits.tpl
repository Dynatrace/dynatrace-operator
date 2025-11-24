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
{{- $number := 0.0 -}}
{{- if not $limit -}}
  {{- $number := -1.0 -}}
{{- else if hasSuffix "Gi" $limit -}}
  {{- $number = float64 (trimSuffix "Gi" $limit) -}}
  {{- $number = mulf $number 1024 -}}
{{- else if hasSuffix "G" $limit -}}
  {{- $number = float64 (trimSuffix "G" $limit) -}}
  {{- $number = mulf $number 1000 -}}
  {{- $number = ceil (mulf 0.931323 $number) -}}
{{- else if hasSuffix "Mi" $limit -}}
  {{- $number = float64 (trimSuffix "Mi" $limit) -}}
{{- else if hasSuffix "M" $limit -}}
  {{- $number = float64 (trimSuffix "M" $limit) -}}
  {{- $number = ceil (mulf 0.953674 $number) -}}
{{- end -}}
{{- if gt $number 0.0 -}}
- name: GOMEMLIMIT
  value: {{ printf "%dMiB" (int64 (ceil (mulf 0.9 $number))) }}
{{- end -}}
{{- end -}}
