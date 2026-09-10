apiVersion: v1
kind: Secret
metadata:
  name: {{ .Release.Name }}
  annotations:
    kubernetes.io/service-account.name: {{ include "common.serviceAccountName" . }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
type: kubernetes.io/service-account-token
