{{- $fullName := include "common.fullname" . -}}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-create-site
  labels:
    {{- include "common.labels" . | nindent 4 }}
    group: {{ .Release.Name }}
    w7.cc/group-name: {{ .Release.Name }}
    w7.cc/job-source: environment-site
  annotations:
    helm.sh/hook: post-install,post-upgrade
    helm.sh/hook-weight: "10"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 3
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        {{- include "common.selectorLabels" . | nindent 8 }}
        group: {{ .Release.Name }}
        w7.cc/group-name: {{ .Release.Name }}
        w7.cc/job-source: environment-site
    spec:
      restartPolicy: Never
      containers:
        - name: create-site
          image: {{ .Values.environment.site.image | quote }}
          command: ["/bin/sh", "-c"]
          args:
            - |-
              set -eu
              PANEL_URL={{ dig "panel" "innerUrl" "" .Values.global | quote }}
              PANEL_TOKEN={{ dig "panel" "panelRealToken" "" .Values.global | quote }}
              NAMESPACE={{ .Release.Namespace | quote }}

              k8s_get() {
                curl -fsSk \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  "https://kubernetes.default.svc$1"
              }

              panel_post() {
                curl -fsSk -X POST \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  -H "Content-Type: application/json" \
                  -d "$2" \
                  "$PANEL_URL$1"
              }

              /home/rangine create:site \
                --app_name={{ .Values.app.identify | quote }} \
                --title={{ .Values.environment.site.title | quote }} \
                --language={{ .Values.environment.site.language | quote }} \
                --version={{ first (splitList "|" (.Values.IMAGE_VERSION | default "")) | quote }} \
                --domain={{ .Values.DOMAIN_URL | quote }} \
                --k8s-app-name={{ .Release.Name | quote }} \
                --k8s-env-app-name={{ $fullName | quote }} \
                --nginx-vhost-template-base64={{ .Values.environment.site.nginxVhostTemplate | b64enc | quote }} \
                --token={{ dig "panel" "panelAccessToken" "" .Values.global | quote }}

              selector="app.kubernetes.io%2Finstance%3Dw7-sitemanager%2Capp.kubernetes.io%2Fname%3Dsite-manager-nginx"
              pods_json=$(k8s_get "/api/v1/namespaces/$NAMESPACE/pods?labelSelector=$selector")
              pod_names=$(printf '%s' "$pods_json" | jq -c '[.items[]? | select(.status.phase == "Running") | .metadata.name]')
              if [ "$(printf '%s' "$pod_names" | jq 'length')" = "0" ]; then
                echo "no running site-manager nginx pod" >&2
                exit 1
              fi
              exec_json=$(jq -n \
                --arg namespace "$NAMESPACE" \
                --argjson podNames "$pod_names" \
                '{
                  namespace: $namespace,
                  podNames: $podNames,
                  containerName: "site-manager-nginx",
                  command: ["nginx", "-s", "reload"],
                  tty: false
                }')
              panel_post "/panel-api/v1/exec-all" "$exec_json" >/dev/null
