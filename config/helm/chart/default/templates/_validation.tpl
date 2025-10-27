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
