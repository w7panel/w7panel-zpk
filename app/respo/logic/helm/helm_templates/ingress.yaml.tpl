{{- if .Values.ingress.__INGRESS_NAME__.enabled -}}
{{- $fullName := include "common.fullname" . -}}
{{- $appName := .Values.global.name -}}
{{- $releaseName := .Release.Name -}}
{{- $domain := .Values.DOMAIN_URL -}}
{{- $backends := include "common.runtimeIngressBackends" (dict "root" . "name" (print $fullName "-__INGRESS_NAME__")) | fromJson -}}
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
  {{- $annotations := deepCopy (.Values.ingress.__INGRESS_NAME__.annotations | default dict) }}
  {{- if $backends }}
    {{- $_ := set $annotations "w7.cc/ingress-backends" (toJson $backends) }}
  {{- end }}
  {{- with $annotations }}
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
          {{- $backendName := $fullName -}}
          {{- if ne .backend.service.name "self" -}}
            {{- $backendName = tpl .backend.service.name $ -}}
          {{- end -}}
          {{- $backend := get $backends $backendName | default (dict "name" $backendName "port" .backend.service.port) }}
          - path: {{ .path }}
            {{- if and .pathType (semverCompare ">=1.18-0" $.Capabilities.KubeVersion.GitVersion) }}
            pathType: {{ .pathType }}
            {{- end }}
            backend:
              {{- if semverCompare ">=1.19-0" $.Capabilities.KubeVersion.GitVersion }}
              service:
                name: {{ $backend.name | quote }}
                port:
                  {{- toYaml $backend.port | nindent 18 }}
              {{- else }}
              serviceName: {{ $backend.name | quote }}
              servicePort: {{ if hasKey $backend.port "name" }}{{ $backend.port.name | quote }}{{ else }}{{ $backend.port.number }}{{ end }}
              {{- end }}
          {{- end }}
    {{- end }}
{{- end }}
