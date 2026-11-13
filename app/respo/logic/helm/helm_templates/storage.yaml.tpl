{{- $storage := .Values.storage | default dict -}}
{{- if and (eq ($storage.mode | default "") "device") (ne (int .Values.replicas) 0) -}}
{{- $name := include "common.storageClaimName" . -}}
{{- $root := . -}}
{{- $needsClaim := false -}}
{{- range $volume := .Values.volumes -}}
{{- if and $volume.persistentVolumeClaim (empty $volume.persistentVolumeClaim.claimName) -}}
{{- $claim := include "common.resolveStorageClaim" (dict "root" $root "name" $volume.name "claim" $name) -}}
{{- if eq $claim $name -}}
{{- $needsClaim = true -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if $needsClaim -}}
{{- $existing := lookup "v1" "PersistentVolumeClaim" .Release.Namespace $name -}}
{{- if not $existing }}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ $name | quote }}
  labels:
    w7.cc/storage-owner: {{ .Release.Name | quote }}
  annotations:
    helm.sh/resource-policy: keep
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-100"
spec:
  accessModes:
    - {{ .Values.storage.accessMode | default "ReadWriteOnce" | quote }}
  {{- with .Values.storage.className }}
  storageClassName: {{ . | quote }}
  {{- end }}
  resources:
    requests:
      storage: {{ .Values.storage.size | default "1Gi" | quote }}
{{- else if ne (index ($existing.metadata.labels | default dict) "w7.cc/storage-owner" | default "") .Release.Name -}}
{{- fail "同名 PVC 不属于当前应用，不能作为独立存储使用" -}}
{{- end }}
{{- end }}
{{- end }}
