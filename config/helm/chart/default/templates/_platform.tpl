# Copyright Dynatrace LLC
# SPDX-License-Identifier: Apache-2.0

{{/*
Auto-detect the platform (if not set), according to the available APIVersions
*/}}
{{- define "dynatrace-operator.platform" -}}
    {{- if .Values.platform}}
        {{- printf .Values.platform -}}
    {{- else if .Capabilities.APIVersions.Has "security.openshift.io/v1" }}
        {{- printf "openshift" -}}
    {{- else }}
        {{- printf "kubernetes" -}}
    {{- end -}}
{{- end }}

{{/*
Auto-detect whether or not we need allowlisting for logagent and csi-driver
*/}}
{{- define "dynatrace-operator.needAutopilotAllowlisting" -}}
    {{- if .Capabilities.APIVersions.Has "auto.gke.io/v1/AllowlistSynchronizer" }}
        {{- printf "true" -}}
    {{- end -}}
{{- end }}

{{/*
Determine if RBAC should use KubeletFineGrainedAuthz (nodes/pods) instead of nodes/proxy.
Auto-enabled on k8s >= 1.36 (KubeletFineGrainedAuthz GA). GKE Autopilot on older k8s also
requires nodes/pods (nodes/proxy is forbidden) but cannot be reliably detected from Helm
capabilities, so it must be enabled there via the override below.
Override with .Values.rbac.kubeletFineGrainedAuthz: true/false
*/}}
{{- define "dynatrace-operator.kubeletFineGrainedAuthz" -}}
    {{- if kindIs "bool" .Values.rbac.kubeletFineGrainedAuthz -}}
        {{- if .Values.rbac.kubeletFineGrainedAuthz -}}
            {{- "true" -}}
        {{- end -}}
    {{- else -}}
        {{- $kubeVersion := semver .Capabilities.KubeVersion.Version -}}
        {{- /* Only accept release versions (e.g. v1.36.0) or GKE versions (e.g. v1.36.1-gke.1495002) */ -}}
        {{- if or (not $kubeVersion.Prerelease) (hasPrefix "gke." $kubeVersion.Prerelease) -}}
            {{- $coreVersion := printf "%d.%d.%d" $kubeVersion.Major $kubeVersion.Minor $kubeVersion.Patch -}}
            {{- if semverCompare ">=1.36.0" $coreVersion -}}
                {{- "true" -}}
            {{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{/*
Set install source how the Operator was installed
*/}}
{{- define "dynatrace-operator.installSource" -}}
    {{- if .Values.olm }}
      {{- printf "operatorhub" -}}
    {{- else if .Values.manifests }}
      {{- printf "manifest" -}}
    {{- else if (and (.Values.platform) (not (has .Values.platform (list "kubernetes" "openshift")))) }}
        {{- printf .Values.platform -}}
    {{- else }}
        {{- printf "helm" -}}
    {{- end -}}
{{- end }}

{{/*
Exclude Kubernetes manifest not running on OLM
*/}}
{{- define "dynatrace-operator.openshiftOrOlm" -}}
{{- if or (eq (include "dynatrace-operator.platform" .) "openshift") (.Values.olm) -}}
    {{ default "true" }}
{{- end -}}
{{- end -}}

{{- define "dynatrace-operator.nodeAffinity" -}}
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/arch
              operator: In
              values:
                - amd64
                - arm64
                - ppc64le
                - s390x
            - key: kubernetes.io/os
              operator: In
              values:
                - linux
{{- end -}}

{{- define "dynatrace-operator.defaultTolerations" -}}
- key: kubernetes.io/arch
  value: arm64
  effect: NoSchedule
- key: kubernetes.io/arch
  value: amd64
  effect: NoSchedule
- key: kubernetes.io/arch
  value: ppc64le
  effect: NoSchedule
- key: kubernetes.io/arch
  value: s390x
  effect: NoSchedule
{{- end -}}
