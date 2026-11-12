apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ .Release.Name }}
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: [{{ printf "%s-results" .Release.Name | quote }}]
    verbs: ["get", "patch"]
{{- range .Values.tasks }}
{{- if and (index $.Values.selected .id) (eq .action "deploy") (not (index $.Values.remote .id)) }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ $.Release.Name }}-{{ .id }}
  namespace: {{ .k8s_namespace | quote }}
rules:
  - apiGroups: ["apps"]
    resources: [{{ .k8s_controller_type | quote }}]
    resourceNames: [{{ .k8s_app_name | quote }}]
    verbs: ["get", "patch"]
{{- end }}
{{- end }}
