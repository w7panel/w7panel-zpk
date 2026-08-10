{{- define "__cur__.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name }}-{{ $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- $fullName := include "__cur__.fullname" . -}}
{{- $sidecarContainers := include "w7panel.sidecars.containers" . | fromYamlArray | default list -}}
{{- $sidecarInitContainers := include "w7panel.sidecars.initContainers" . | fromYamlArray | default list -}}
{{- $sidecarVolumes := include "w7panel.sidecars.volumes" . | fromYamlArray | default list -}}
{{- $targetPodAnnotations := include "w7panel.podAnnotations" . | fromYaml | default dict -}}

apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-create-register-site
  labels:
    group: {{ .Release.Name }}
    w7.cc/group-name: {{ .Release.Name }}
    w7.cc/job-source: appgroup
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-10"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 2
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        group: {{ .Release.Name }}
        w7.cc/group-name: {{ .Release.Name }}
        w7.cc/job-source: appgroup
      {{- if .Values.podAnnotations }}
      annotations:
        {{- toYaml .Values.podAnnotations | nindent 8 }}
        {{- if .Values.annotations }}
        {{- toYaml .Values.annotations | nindent 8 }}
        {{- end }}
      {{- else if .Values.annotations }}
      annotations:
        {{- toYaml .Values.annotations | nindent 8 }}
      {{- end }}
    spec:
      restartPolicy: Never
      containers:
        - name: create-site-job
          image: zpk.w7.cc/public/site-manager:v1.2.18
          command:
            - sh
            - -c
            - |-
              set -eu

              PANEL_URL="{{ .Values.global.panel.innerUrl }}"
              PANEL_TOKEN="{{ .Values.global.panel.panelRealToken }}"
              PANEL_ACCESSTOKEN="{{ .Values.global.panel.panelAccessToken }}"
              NAMESPACE="{{ .Release.Namespace }}"
              OPERATION='{{ ternary "upgrade" "install" .Release.IsUpgrade }}'
              ENV_TITLE=__ENV_TITLE__
              ENV_GROUP=__ENV_GROUP__
              ENV_LANGUAGE=__ENV_LANGUAGE__
              ENV_VERSION=__ENV_VERSION__
              DOMAIN='{{ .Values.DOMAIN_URL }}'
              ENABLE_SSL='{{ default false .Values.ingressForceHttps }}'
              APP_IDENTIFY=__APP_IDENTIFY__
              SITE_K8S_APP='{{ $fullName }}'
              NEW_ENV_K8S_APP='__NEW_ENV_K8S_APP__'
              CODE_DOWNLOAD_URL=__CODE_DOWNLOAD_URL__
              STATE_CONFIG='{{ $fullName }}-site-state'
              SIDECAR_CONTAINERS_B64='{{ if $sidecarContainers }}{{ $sidecarContainers | toJson | b64enc }}{{ end }}'
              SIDECAR_INIT_CONTAINERS_B64='{{ if $sidecarInitContainers }}{{ $sidecarInitContainers | toJson | b64enc }}{{ end }}'
              SIDECAR_VOLUMES_B64='{{ if $sidecarVolumes }}{{ $sidecarVolumes | toJson | b64enc }}{{ end }}'
              POD_ANNOTATIONS_B64='{{ if $targetPodAnnotations }}{{ $targetPodAnnotations | toJson | b64enc }}{{ end }}'

              panel_safe_name() {
                echo "$1" | tr '_' '-'
              }

              k8s_get() {
                curl -fsSk \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  "https://kubernetes.default.svc$1"
              }

              k8s_post() {
                path="$1"
                data="$2"
                curl -fsSk -X POST \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  -H "Content-Type: application/json" \
                  -d "$data" \
                  "https://kubernetes.default.svc$path"
              }

              k8s_patch() {
                path="$1"
                data="$2"
                curl -fsSk -X PATCH \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  -H "Content-Type: application/merge-patch+json" \
                  -d "$data" \
                  "https://kubernetes.default.svc$path"
              }

              k8s_delete() {
                path="$1"
                curl -fsSk -X DELETE \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  "https://kubernetes.default.svc$path"
              }

              k8s_delete "/api/v1/namespaces/$NAMESPACE/configmaps/$STATE_CONFIG" >/dev/null 2>&1 || true

              query_deploy() {
                deploy_name="$1"
                k8s_get "/apis/apps/v1/namespaces/$NAMESPACE/deployments/$(panel_safe_name "$deploy_name")"
              }

              get_site_env_app_name() {
                site_info=$(/home/rangine info:site \
                  --token="$PANEL_ACCESSTOKEN" \
                  --domain="$DOMAIN" \
                  -f /home/config.yaml 2>/dev/null || true)
                printf '%s' "$site_info" | jq -r '.site_environment.app_name // empty' 2>/dev/null || true
              }

              save_state() {
                target_env_deploy=$(get_site_env_app_name)
                if [ -z "$target_env_deploy" ]; then
                  target_env_deploy="$NEW_ENV_K8S_APP"
                fi
                target_deploy_json=$(query_deploy "$target_env_deploy")
                target_env_image=$(printf '%s' "$target_deploy_json" | jq -r '.spec.template.spec.containers[0].image // ""')
                target_env_volumes=$(printf '%s' "$target_deploy_json" | jq -c '.spec.template.spec.volumes // []')
                target_env_volume_mounts=$(printf '%s' "$target_deploy_json" | jq -c '.spec.template.spec.containers[0].volumeMounts // []')
                target_env_affinity=$(printf '%s' "$target_deploy_json" | jq -c '.spec.template.spec.affinity // {}')
                target_env_resources=$(printf '%s' "$target_deploy_json" | jq -c '.spec.template.spec.containers[0].resources // {}')
                target_env_security_context=$(printf '%s' "$target_deploy_json" | jq -c '.spec.template.spec.containers[0].securityContext // {}')
                target_env_image_pull_policy=$(printf '%s' "$target_deploy_json" | jq -r '.spec.template.spec.containers[0].imagePullPolicy // "IfNotPresent"')

                state_json=$(jq -n \
                  --arg name "$STATE_CONFIG" \
                  --arg targetEnvDeploy "$target_env_deploy" \
                  --arg targetEnvImage "$target_env_image" \
                  --arg targetEnvVolumes "$target_env_volumes" \
                  --arg targetEnvVolumeMounts "$target_env_volume_mounts" \
                  --arg targetEnvAffinity "$target_env_affinity" \
                  --arg targetEnvResources "$target_env_resources" \
                  --arg targetEnvSecurityContext "$target_env_security_context" \
                  --arg targetEnvImagePullPolicy "$target_env_image_pull_policy" \
                  --arg operation "$OPERATION" \
                  --arg domain "$DOMAIN" \
                  --arg namespace "$NAMESPACE" \
                  '{
                    apiVersion: "v1",
                    kind: "ConfigMap",
                    metadata: {
                      name: $name,
                      namespace: $namespace
                    },
                    data: {
                      target_env_deploy: $targetEnvDeploy,
                      target_env_image: $targetEnvImage,
                      target_env_volumes: $targetEnvVolumes,
                      target_env_volume_mounts: $targetEnvVolumeMounts,
                      target_env_affinity: $targetEnvAffinity,
                      target_env_resources: $targetEnvResources,
                      target_env_security_context: $targetEnvSecurityContext,
                      target_env_image_pull_policy: $targetEnvImagePullPolicy,
                      operation: $operation,
                      domain: $domain
                    }
                  }')
                k8s_post "/api/v1/namespaces/$NAMESPACE/configmaps" "$state_json" >/dev/null \
                  || k8s_patch "/api/v1/namespaces/$NAMESPACE/configmaps/$STATE_CONFIG" "$state_json" >/dev/null
              }

              /home/rangine provision:site \
                --panel-url="$PANEL_URL" \
                --panel-token="$PANEL_TOKEN" \
                --panel-access-token="$PANEL_ACCESSTOKEN" \
                --namespace="$NAMESPACE" \
                --operation="$OPERATION" \
                --release="{{ .Release.Name }}" \
                --title="$ENV_TITLE" \
                --name="$ENV_GROUP" \
                --language="$ENV_LANGUAGE" \
                --version="$ENV_VERSION" \
                --domain="$DOMAIN" \
                --ssl="$ENABLE_SSL" \
                --code-download-url="$CODE_DOWNLOAD_URL" \
                --app-name="$APP_IDENTIFY" \
                --site-k8s-app-name="$SITE_K8S_APP" \
                --target-env-app-name="$NEW_ENV_K8S_APP" \
                --sidecar-containers="$SIDECAR_CONTAINERS_B64" \
                --sidecar-init-containers="$SIDECAR_INIT_CONTAINERS_B64" \
                --sidecar-volumes="$SIDECAR_VOLUMES_B64" \
                --pod-annotations="$POD_ANNOTATIONS_B64" \
                -f /home/config.yaml

              save_state
