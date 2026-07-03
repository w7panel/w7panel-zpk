apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Release.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
subjects:
- kind: ServiceAccount
  name: {{ include "common.serviceAccountName" . }}
  namespace: {{ .Release.Namespace }}
roleRef:
  kind: ClusterRole
  name: {{ .Release.Name }}
  apiGroup: rbac.authorization.k8s.io
