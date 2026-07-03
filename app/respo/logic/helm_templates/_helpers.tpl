{{/*
Expand the name of the chart.
*/}}
{{- define "common.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "common.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "common.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "common.labels" -}}
helm.sh/chart: {{ include "common.chart" . }}
{{ include "common.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
w7.cc/identifie: __IDENTIFY__
{{- end }}

{{/*
Selector labels
*/}}
{{- define "common.selectorLabels" -}}
app: {{ include "common.fullname" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "common.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "common.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service to use
*/}}
{{- define "common.serviceName" -}}
{{ include "common.fullname" . }}
{{- end }}

{{/*
Create pull secrets
*/}}
{{- define "common.pullSecrets" -}}
{{- if .Values.image.pullSecrets }}
imagePullSecrets:
{{ toYaml .Values.image.pullSecrets | indent 2 }}
{{- end }}
{{- end }}


{{/*
Renders a list of volumes with proper handling of PVC and global PVC_NAME.
Usage:
  {{ include "common.volumesToYaml" (dict "root" $ "volumes" .Values.volumes) }}
*/}}
{{- define "common.volumesToYaml" }}
{{- $root := .root }}
{{- $volumes := .volumes }}

{{- if $volumes }}
{{- range $vol := $volumes }}
- name: {{ $vol.name | quote }}
  {{- if $vol.persistentVolumeClaim }}
  persistentVolumeClaim:
    claimName: {{ coalesce $vol.persistentVolumeClaim.claimName $root.Values.PVC_NAME | quote }}
  {{- else }}
    {{- /* Reconstruct the volume spec without 'name' and 'persistentVolumeClaim' */}}
    {{- $spec := dict }}
    {{- range $key, $value := $vol }}
      {{- if and (ne $key "name") (ne $key "persistentVolumeClaim") }}
        {{- $_ := set $spec $key $value }}
      {{- end }}
    {{- end }}
    {{- if empty $spec }}
  emptyDir: {}
    {{- else }}
    {{- toYaml $spec | nindent 2 }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Renders only volumeMounts backed by pod volumes.
Used by Job templates, which cannot consume StatefulSet volumeClaimTemplates.
*/}}
{{- define "common.jobVolumeMountsToYaml" }}
{{- $root := .root }}
{{- $mounts := .mounts }}
{{- $allowed := dict }}
{{- range $vol := $root.Values.volumes }}
{{- if $vol.name }}
{{- $_ := set $allowed $vol.name true }}
{{- end }}
{{- end }}
{{- $validMounts := list }}
{{- range $mount := $mounts }}
{{- if hasKey $allowed $mount.name }}
{{- $validMounts = append $validMounts $mount }}
{{- end }}
{{- end }}
{{- if $validMounts }}
{{- tpl (toYaml $validMounts) $root }}
{{- end }}
{{- end }}
