{{- $runtimeConfig := include "common.runtimeConfig" . | fromJson -}}
{{- $servicePorts := .Values.service.ports -}}
{{- if hasKey $runtimeConfig "servicePorts" -}}
  {{- $servicePorts = $runtimeConfig.servicePorts -}}
{{- end -}}
{{- if $servicePorts }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "common.fullname" . }}
  annotations:
    meta.helm.sh/release-name: {{ .Release.Name | quote }}
    meta.helm.sh/release-namespace: {{ .Release.Namespace | quote }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  type: {{ $runtimeConfig.serviceType | default .Values.service.type }}
  selector:
    {{- include "common.selectorLabels" . | nindent 4 }}
  ports:
    {{- range $servicePorts }}
    - port: {{ .port }}
      targetPort: {{ .targetPort }}
      protocol: {{ .protocol }}
      name: {{ .name }}
      {{- with .appProtocol }}
      appProtocol: {{ . | quote }}
      {{- end }}
    {{- end }}
{{- end }}
