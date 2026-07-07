{{- if .Values.ingress.__INGRESS_NAME__.enabled -}}
{{- $fullName := include "common.fullname" . -}}
{{- $appName := .Values.global.name -}}
{{- $releaseName := .Release.Name -}}
{{- $domain := .Values.DOMAIN_URL -}}
{{- if and .Values.ingress.__INGRESS_NAME__.ingressClassName (not (semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion)) }}
  {{- if not (hasKey .Values.ingress.__INGRESS_NAME__.annotations "kubernetes.io/ingress.class") }}
  {{- $_ := set .Values.ingress.__INGRESS_NAME__.annotations "kubernetes.io/ingress.class" .Values.ingress.__INGRESS_NAME__.ingressClassName}}
  {{- end }}
{{- end }}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1beta1
{{- else -}}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ $fullName }}-__INGRESS_NAME__
  labels:
    {{- include "common.labels" . | nindent 4 }}
    group: {{ $releaseName }}
    w7.cc/group-name: {{ $releaseName }}
    __PARENT_INGRESS_LABEL__
  annotations:
  {{- with .Values.ingress.__INGRESS_NAME__.annotations }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
    {{- if .Values.ingressForceHttps }}
    cert-manager.io/cluster-issuer: w7-letsencrypt-prod
    cert-manager.io/renew-before: 30m
    {{- end }}
    kubernetes.io/ingress.class: higress
spec:
  {{- if .Values.ingressForceHttps }}
  tls:
    - hosts:
        {{- range .Values.ingress.__INGRESS_NAME__.hosts }}
        - {{ coalesce .host $domain | quote }}
        {{- end }}
      secretName: {{ $fullName }}-__INGRESS_NAME__-tls-secret
  {{- end }}
  rules:
    {{- range .Values.ingress.__INGRESS_NAME__.hosts }}
    - host: {{ coalesce .host $domain | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            {{- if and .pathType (semverCompare ">=1.18-0" $.Capabilities.KubeVersion.GitVersion) }}
            pathType: {{ .pathType }}
            {{- end }}
            backend:
              {{- if semverCompare ">=1.19-0" $.Capabilities.KubeVersion.GitVersion }}
              service:
                {{- if eq .backend.service.name "self" }}
                name: {{ $fullName }}
                {{- else }}
                name: {{ tpl .backend.service.name $ }}
                {{- end }}
                port:
                  number: {{ .backend.service.port.number }}
              {{- else }}
              {{- if eq .backend.service.name "self" }}
              serviceName: {{ $fullName }}
              {{- else }}
              serviceName: {{ tpl .backend.service.name $ }}
              {{- end }}
              servicePort: {{ .backend.service.port.number }}
              {{- end }}
          {{- end }}
    {{- end }}
{{- end }}
