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
Render pod annotations from values. Annotation values may contain Helm
expressions (for example, a release revision marker injected for an
environment's imported Nginx child), so evaluate every value uniformly
instead of special-casing a particular annotation key.
*/}}
{{- define "w7panel.podAnnotations" -}}
{{- $root := . -}}
{{- $annotations := dict -}}
{{- range $key, $value := ($root.Values.podAnnotations | default dict) -}}
  {{- $_ := set $annotations $key (tpl (toString $value) $root) -}}
{{- end -}}
{{- range $key, $value := ($root.Values.annotations | default dict) -}}
  {{- $_ := set $annotations $key (tpl (toString $value) $root) -}}
{{- end -}}
{{- $sidecarAnnotations := include "w7panel.sidecars.podAnnotations" $root | fromYaml | default dict -}}
{{- range $key, $value := $sidecarAnnotations -}}
  {{- $_ := set $annotations $key (tpl (toString $value) $root) -}}
{{- end -}}
{{- if $annotations -}}
{{- toYaml $annotations -}}
{{- end -}}
{{- end -}}

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
Keep RANDOM_DIR mounts stable across upgrades. Existing workloads are the
source of truth; new installations use the original deterministic fallback.
*/}}
{{- define "common.resolveStableSubPath" -}}
{{- $root := .root -}}
{{- $containerName := .containerName -}}
{{- $volumeName := .volumeName -}}
{{- $mountPath := .mountPath -}}
{{- $fallback := printf "%s|%s|%s|%s|%s|%s" $root.Release.Name $root.Release.Namespace $root.Chart.Name $containerName $volumeName $mountPath | sha256sum | trunc 12 -}}
{{- $workloadName := include "common.fullname" $root -}}
{{- $existing := lookup "apps/v1" "Deployment" $root.Release.Namespace $workloadName -}}
{{- if not $existing -}}
  {{- $existing = lookup "apps/v1" "StatefulSet" $root.Release.Namespace $workloadName -}}
{{- end -}}
{{- if not $existing -}}
  {{- $existing = lookup "apps/v1" "DaemonSet" $root.Release.Namespace $workloadName -}}
{{- end -}}
{{- $existingSubPath := "" -}}
{{- if $existing -}}
  {{- $spec := default (dict) (index $existing "spec") -}}
  {{- $template := default (dict) (index $spec "template") -}}
  {{- $podSpec := default (dict) (index $template "spec") -}}
  {{- $containerGroups := list (default (list) (index $podSpec "containers")) (default (list) (index $podSpec "initContainers")) -}}
  {{- range $containers := $containerGroups -}}
    {{- range $container := $containers -}}
      {{- if eq (default "" $container.name) $containerName -}}
        {{- range $mount := (default (list) $container.volumeMounts) -}}
          {{- if and (eq $existingSubPath "") (eq (default "" $mount.name) $volumeName) (eq (default "" $mount.mountPath) $mountPath) (ne (default "" $mount.subPath) "") -}}
            {{- $existingSubPath = $mount.subPath -}}
          {{- end -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- default $fallback $existingSubPath -}}
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
    {{ $claimName := dig "persistentVolumeClaim" "claimName" "" $vol }}
    {{ $claim := toString $claimName }}
    {{ if eq $claim "" }}
      {{ $claim = toString (get $root.Values "PVC_NAME") }}
    {{ end }}
    claimName: {{ tpl $claim $root | quote }}
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
