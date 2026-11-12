{{- range .Values.tasks }}
{{- if and (index $.Values.selected .id) (eq .action "deploy") (index $.Values.remote .id) }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ $.Release.Name }}-{{ .id }}
type: Opaque
data:
  config: {{ required "目标集群 KUBECONFIG 不能为空" (index $.Values.kubeconfigs .id) | b64enc | quote }}
{{- end }}
{{- end }}

{{- range .Values.tasks }}
{{- if and (index $.Values.selected .id) (eq .action "build") }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ $.Release.Name }}-auth-{{ .id }}
type: Opaque
data:
  config.json: {{ dict "auths" (dict $.Values.run.host (dict "registrytoken" (required "构建任务镜像仓库令牌不能为空" $.Values.registryToken))) | toJson | b64enc | quote }}
{{- end }}
{{- end }}
