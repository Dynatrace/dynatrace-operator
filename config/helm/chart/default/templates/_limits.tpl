# Copyright Dynatrace LLC
# SPDX-License-Identifier: Apache-2.0

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
