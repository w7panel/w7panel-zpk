apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ .Release.Name }}
subjects:
  - kind: ServiceAccount
    name: {{ .Release.Name }}
    namespace: {{ .Release.Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ .Release.Name }}
{{- range .Values.tasks }}
{{- if and (index $.Values.selected .id) (eq .action "deploy") (not (index $.Values.remote .id)) }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ $.Release.Name }}-{{ .id }}
  namespace: {{ .k8s_namespace | quote }}
subjects:
  - kind: ServiceAccount
    name: {{ $.Release.Name }}
    namespace: {{ $.Release.Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ $.Release.Name }}-{{ .id }}
{{- end }}
{{- end }}
