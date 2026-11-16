{{- $config := .Values.ingress.__INGRESS_NAME__ -}}
{{- $group := get (.Values.domainGroups | default dict) ($config.domainParam | default "") | default dict -}}
{{- $domain := .Values.DOMAIN_URL -}}
{{- $https := .Values.ingressForceHttps -}}
{{- if $config.domainParam -}}
  {{- $domain = $group.host | default "" -}}
  {{- $https = $group.autoSSL | default false -}}
{{- end -}}
{{- if $config.enabled -}}
{{- $fullName := include "common.fullname" . -}}
{{- $existing := lookup "networking.k8s.io/v1" "Ingress" .Release.Namespace (print $fullName "-__INGRESS_NAME__") | default dict -}}
{{- if $config.domainParam -}}
  {{- if not $domain -}}
    {{- $domain = printf "%s.private.cluster.invalid" (printf "%s/%s/%s/%s" .Release.Namespace .Release.Name $fullName $config.domainParam | sha256sum | trunc 24) -}}
  {{- end -}}
{{- end -}}
{{- $annotations := deepCopy ($config.annotations | default dict) -}}
{{- $oldSpec := $existing.spec | default dict -}}
{{- $existingDomain := "" -}}
{{- if $oldSpec.rules -}}
  {{- $existingDomain = (first $oldSpec.rules).host -}}
  {{- $domain = $existingDomain -}}
{{- end -}}
{{- $requireHTTPS := eq (toString (get $annotations "w7.cc/require-https")) "true" -}}
{{- $https = and (or $https $requireHTTPS) (not (hasSuffix ".invalid" $domain)) -}}
{{- $appName := .Values.global.name -}}
{{- $releaseName := .Release.Name -}}
{{- $backends := include "common.runtimeIngressBackends" (dict "root" . "name" (print $fullName "-__INGRESS_NAME__")) | fromJson -}}
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
  {{- if $backends }}
    {{- $_ := set $annotations "w7.cc/ingress-backends" (toJson $backends) }}
  {{- end }}
  {{- with $annotations }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
    kubernetes.io/ingress.class: higress
spec:
  {{- if $https }}
  tls:
    - hosts:
        {{- range $config.hosts }}
        - {{ coalesce $existingDomain .host $domain | quote }}
        {{- end }}
      secretName: {{ $fullName }}-__INGRESS_NAME__-tls-secret
  {{- end }}
  rules:
    {{- range $config.hosts }}
    - host: {{ coalesce $existingDomain .host $domain | quote }}
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
