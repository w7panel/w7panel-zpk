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
  config.json: {{ dict "auths" (dict $.Values.run.host (dict "auth" (printf "%s:%s" (required "镜像仓库用户名不能为空" $.Values.registryUsername) (required "镜像仓库密码不能为空" $.Values.registryPassword) | b64enc))) | toJson | b64enc | quote }}
{{- end }}
{{- end }}
