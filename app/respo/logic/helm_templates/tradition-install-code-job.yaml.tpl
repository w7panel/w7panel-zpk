{{- $fullName := include "common.fullname" . -}}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-install-code
  labels:
    group: {{ .Release.Name }}
    w7.cc/group-name: {{ .Release.Name }}
    w7.cc/job-source: appgroup
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
        group: {{ .Release.Name }}
        w7.cc/group-name: {{ .Release.Name }}
        w7.cc/job-source: tradition-install
    spec:
      restartPolicy: Never
      {{- with .Values.tradition.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      volumes:
        - name: "site-storage"
          persistentVolumeClaim:
            claimName: "w7-sitemanager-site-manager"
      containers:
        - name: install-code
          image: busybox:1.36.1
          command: ["/bin/sh", "-c"]
          args:
            - |-
              set -eu
              : "${CODE_INSTALL_DIRECTORY:?CODE_INSTALL_DIRECTORY is required}"
              : "${SITE_ROOT:?SITE_ROOT is required}"
              : "${CODE_PACKAGE_URL:?CODE_PACKAGE_URL is required}"

              case "$CODE_INSTALL_DIRECTORY" in
                */*|*..*|*[!A-Za-z0-9._-]*)
                  echo "invalid code install directory: $CODE_INSTALL_DIRECTORY" >&2
                  exit 1
                  ;;
              esac

              mkdir -p "$SITE_ROOT"
              tmp_zip="$(mktemp /tmp/tradition-code.XXXXXX.zip)"
              trap 'rm -f "$tmp_zip"' EXIT
              wget -q -O "$tmp_zip" "$CODE_PACKAGE_URL"
              unzip -oq "$tmp_zip" -d "$SITE_ROOT"
          env:
            - name: CODE_INSTALL_DIRECTORY
              value: {{ .Values.CODE_INSTALL_DIRECTORY | quote }}
            - name: SITE_ROOT
              value: {{ printf "/www/wwwroot/%s" .Values.CODE_INSTALL_DIRECTORY | quote }}
            - name: CODE_PACKAGE_URL
              value: {{ .Values.tradition.codePackageUrl | quote }}
          volumeMounts:
            - name: "site-storage"
              mountPath: {{ printf "/www/wwwroot/%s" .Values.CODE_INSTALL_DIRECTORY | quote }}
              subPath: {{ printf "nginx-web-dir/%s" .Values.CODE_INSTALL_DIRECTORY | quote }}
