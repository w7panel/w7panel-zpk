{{- define "document.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "document.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "document.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "document.selectorLabels" -}}
app.kubernetes.io/name: {{ include "document.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "document.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
{{ include "document.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "document.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "document.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "document.storageClaimName" -}}
{{- default (include "document.fullname" .) .Values.persistence.existingClaim -}}
{{- end -}}

{{- define "document.podAnnotations" -}}
{{- $annotations := include "w7panel.podAnnotations" . | fromYaml | default dict -}}
{{- if .Values.w7panelSidecars -}}
{{- $_ := set $annotations "w7.cc/inject-root-ca" "true" -}}
{{- end -}}
{{- if $annotations -}}
{{ toYaml $annotations }}
{{- end -}}
{{- end -}}
