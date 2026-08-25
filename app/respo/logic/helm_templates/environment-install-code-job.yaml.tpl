{{- $fullName := include "common.fullname" . -}}
{{- if .Values.environment.code.enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-install-code
  labels:
    {{- include "common.labels" . | nindent 4 }}
    group: {{ .Release.Name }}
    w7.cc/group-name: {{ .Release.Name }}
    w7.cc/job-source: environment-code
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-3"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 2
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        {{- include "common.selectorLabels" . | nindent 8 }}
        group: {{ .Release.Name }}
        w7.cc/group-name: {{ .Release.Name }}
        w7.cc/job-source: environment-code
    spec:
      restartPolicy: Never
      {{- with .Values.jobAffinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      volumes:
        {{- include "common.volumesToYaml" (dict "root" . "volumes" .Values.volumes) | nindent 8 }}
      containers:
        - name: install-code
          image: {{ .Values.environment.code.image | quote }}
          imagePullPolicy: IfNotPresent
          command: ["/bin/sh", "-c"]
          args:
            - |-
              set -eu
              : "${DOMAIN_URL:?DOMAIN_URL is required}"
              test -n "$CODE_PACKAGE_URL"
              mkdir -p "$CODE_INSTALL_PATH"
              tmp_zip="$(mktemp /tmp/environment-code.XXXXXX)"
              trap 'rm -f "$tmp_zip"' EXIT
              wget -q -O "$tmp_zip" "$CODE_PACKAGE_URL"
              unzip -oq "$tmp_zip" -d "$CODE_INSTALL_PATH"
          env:
            - name: DOMAIN_URL
              value: {{ .Values.DOMAIN_URL | quote }}
            - name: CODE_PACKAGE_URL
              value: {{ .Values.environment.code.packageUrl | quote }}
            - name: CODE_INSTALL_PATH
              value: {{ printf "/www/wwwroot/%s" .Values.DOMAIN_URL | quote }}
          volumeMounts:
            - name: {{ .Values.environment.code.volumeName | quote }}
              mountPath: {{ printf "/www/wwwroot/%s" .Values.DOMAIN_URL | quote }}
              subPath: {{ printf "nginx-web-dir/%s" .Values.DOMAIN_URL | quote }}
{{- end }}
