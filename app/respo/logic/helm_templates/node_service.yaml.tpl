{{- if .Values.node_service.ports }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "common.fullname" . }}-lb
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  type: {{ .Values.node_service.type }}
  selector:
    {{- include "common.selectorLabels" . | nindent 4 }}
  ports:
    {{- range .Values.node_service.ports }}
    - port: {{ .port }}
      targetPort: {{ .targetPort }}
      protocol: {{ .protocol }}
      name: {{ .name }}
    {{- end }}
{{- end }}
