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

apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $fullName }}-site-shell
  labels:
    group: {{ .Release.Name }}
    w7.cc/job-source: appgroup
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-5"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        group: {{ .Release.Name }}
        w7.cc/job-source: appgroup
      {{- if .Values.podAnnotations }}
      annotations:
        {{- toYaml .Values.podAnnotations | nindent 8 }}
      {{- end }}
    spec:
      restartPolicy: Never
      containers:
        - name: site-shell-job
          image: zpk.w7.cc/public/site-manager:v1.2.12
          command:
            - sh
            - -c
            - |-
              set -eu

              PANEL_URL="{{ .Values.global.panel.innerUrl }}"
              PANEL_TOKEN="{{ .Values.global.panel.panelRealToken }}"
              NAMESPACE="{{ .Release.Namespace }}"
              STATE_CONFIG='{{ $fullName }}-site-state'
              CMD_B64='__CMD_B64__'
              SHELLS_B64='__SHELLS_B64__'
              START_PARAMS_ENV_B64='__START_PARAMS_ENV_B64__'

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
                  -H "Content-Type: application/strategic-merge-patch+json" \
                  -d "$data" \
                  "https://kubernetes.default.svc$path"
              }

              k8s_delete() {
                path="$1"
                curl -fsSk -X DELETE \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  "https://kubernetes.default.svc$path"
              }

              panel_post() {
                path="$1"
                data="$2"
                curl -fsS -X POST \
                  -H "Authorization: Bearer $PANEL_TOKEN" \
                  -H "Content-Type: application/json" \
                  -d "$data" \
                  "$PANEL_URL$path"
              }

              decode_b64_json() {
                val="$1"
                fallback="$2"
                if [ -z "$val" ] || [ "$val" = "{}" ] || [ "$val" = "null" ]; then
                  echo "$fallback"
                  return
                fi
                printf '%s' "$val" | base64 -d
              }

              load_state() {
                start_ts=$(date +%s)
                while true; do
                  state_json=$(k8s_get "/api/v1/namespaces/$NAMESPACE/configmaps/$STATE_CONFIG" 2>/dev/null || true)
                  if printf '%s' "$state_json" | jq -e '.data.target_env_deploy' >/dev/null 2>&1; then
                    printf '%s' "$state_json"
                    return
                  fi
                  now_ts=$(date +%s)
                  if [ $((now_ts - start_ts)) -gt 300 ]; then
                    echo "site state configmap not ready: $STATE_CONFIG" >&2
                    exit 1
                  fi
                  sleep 2
                done
              }

              query_deploy() {
                deploy_name="$1"
                k8s_get "/apis/apps/v1/namespaces/$NAMESPACE/deployments/$(panel_safe_name "$deploy_name")"
              }

              log_shell() {
                printf '[site-shell] %s %s\n' "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" "$*"
              }

              wait_job() {
                job_name="$1"
                safe_job_name=$(panel_safe_name "$job_name")
                start_ts=$(date +%s)
                log_shell "wait job start: name=$safe_job_name"
                while true; do
                  job_json=$(k8s_get "/apis/batch/v1/namespaces/$NAMESPACE/jobs/$safe_job_name")
                  complete=$(printf '%s' "$job_json" | jq -r '[.status.conditions[]? | select(.type=="Complete" and .status=="True")] | length')
                  failed=$(printf '%s' "$job_json" | jq -r '[.status.conditions[]? | select(.type=="Failed" and .status=="True")] | length')
                  if [ "$complete" != "0" ]; then
                    log_shell "wait job complete: name=$safe_job_name"
                    return 0
                  fi
                  if [ "$failed" != "0" ]; then
                    log_shell "wait job failed: name=$safe_job_name"
                    printf '%s\n' "$job_json" | jq -c '.status.conditions // []'
                    return 1
                  fi
                  now_ts=$(date +%s)
                  if [ $((now_ts - start_ts)) -gt 600 ]; then
                    log_shell "wait job timeout: name=$safe_job_name"
                    return 1
                  fi
                  sleep 2
                done
              }

              build_env_array() {
                start_params_json="$1"
                jq -n \
                  --argjson startParams "$start_params_json" \
                  '
                  (
                    $startParams
                    | to_entries
                    | map({name: .key, value: (.value|tostring)})
                  )
                  '
              }

              create_shell_job() {
                shell_type="$1"
                shell_title="$2"
                shell_image="$3"
                shell_body="$4"
                start_params_json="$5"

                env_array=$(build_env_array "$start_params_json")
                job_name="$(printf '%s' "$TARGET_ENV_DEPLOY-$shell_type-$(date +%s)-$RANDOM" | tr '[:upper:]_' '[:lower:]-' | cut -c1-63)"
                container_name="$(printf '%s' "$TARGET_ENV_DEPLOY-shell" | tr '[:upper:]_' '[:lower:]-' | cut -c1-63)"
                job_image="$shell_image"
                if [ -z "$job_image" ] || [ "$job_image" = "null" ]; then
                  job_image="$TARGET_ENV_IMAGE"
                fi
                log_shell "create job start: type=$shell_type title=$shell_title name=$job_name image=$job_image"

                job_json=$(jq -n \
                  --arg jobName "$job_name" \
                  --arg targetApp "$TARGET_ENV_DEPLOY" \
                  --arg fullName "{{ $fullName }}" \
                  --arg releaseName "{{ .Release.Name }}" \
                  --arg containerName "$container_name" \
                  --arg shellType "$shell_type" \
                  --arg shellTitle "$shell_title" \
                  --arg jobImage "$job_image" \
                  --arg shellBody "$shell_body" \
                  --arg imagePullPolicy "$TARGET_ENV_IMAGE_PULL_POLICY" \
                  --arg namespace "$NAMESPACE" \
                  --argjson envArray "$env_array" \
                  --argjson volumes "$TARGET_ENV_VOLUMES" \
                  --argjson volumeMounts "$TARGET_ENV_VOLUME_MOUNTS" \
                  --argjson affinity "$TARGET_ENV_AFFINITY" \
                  --argjson resources "$TARGET_ENV_RESOURCES" \
                  --argjson securityContext "$TARGET_ENV_SECURITY_CONTEXT" \
                  '
                  {
                      apiVersion: "batch/v1",
                      kind: "Job",
                      metadata: {
                        name: $jobName,
                        namespace: $namespace,
                        labels: {
                          app: $targetApp,
                          group: $releaseName,
                          "w7.cc/job-source": "appgroup"
                        },
                        annotations: ({
                          "w7.cc/shell-type": $shellType,
                          "w7.cc/title": $shellTitle,
                          "w7.cc/group-name": $fullName,
                        } + (if $shellType == "custom" then {"w7.cc/custom-hook":"true"} else {} end))
                      },
                      spec: {
                        backoffLimit: 0,
                        ttlSecondsAfterFinished: 60,
                        template: {
                          metadata: {
                            annotations: {
                              "w7.cc/group-name": $fullName,
                            },
                            labels: {
                              app: $jobName,
                              group: $releaseName,
                              "w7.cc/job-source": "tradition-site"
                            }
                          },
                          spec: {
                            restartPolicy: "Never",
                            affinity: $affinity,
                            volumes: $volumes,
                            containers: [
                              {
                                name: $containerName,
                                image: $jobImage,
                                imagePullPolicy: $imagePullPolicy,
                                command: ["/bin/sh", "-c"],
                                args: [$shellBody],
                                env: $envArray,
                                resources: $resources,
                                volumeMounts: $volumeMounts,
                                securityContext: $securityContext
                              }
                            ]
                          }
                        }
                      }
                    }')

                k8s_post "/apis/batch/v1/namespaces/$NAMESPACE/jobs" "$job_json" >/dev/null
                log_shell "create job success: type=$shell_type title=$shell_title name=$job_name"
                wait_job "$job_name"
                log_shell "shell job finished: type=$shell_type title=$shell_title name=$job_name"
              }

              run_restart_patch() {
                if [ -z "$CMD_B64" ] || [ "$CMD_B64" = "{}" ] || [ "$CMD_B64" = "null" ]; then
                  log_shell "skip restart patch: reason=empty-command"
                  return 0
                fi
                cmd_json=$(decode_b64_json "$CMD_B64" "[]")
                if ! printf '%s' "$cmd_json" | jq -e 'type == "array"' >/dev/null 2>&1; then
                  log_shell "skip restart patch: reason=invalid-command-json"
                  return 0
                fi
                cmd_json=$(printf '%s' "$cmd_json" | jq -c 'map(select(type == "string" and test("\\S")))')
                if [ "$(printf '%s' "$cmd_json" | jq 'length')" = "0" ]; then
                  log_shell "skip restart patch: reason=empty-command"
                  return 0
                fi
                patch_json=$(jq -n \
                  --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                  --arg deployName "$TARGET_ENV_DEPLOY" \
                  --argjson cmd "$cmd_json" \
                  '{
                    spec: {
                      template: {
                        metadata: {
                          annotations: {
                            "kubectl.kubernetes.io/restartedAt": $ts
                          }
                        },
                        spec: {
                          containers: [
                            {
                              name: $deployName,
                              command: $cmd
                            }
                          ]
                        }
                      }
                    }
                  }')
                k8s_patch "/apis/apps/v1/namespaces/$NAMESPACE/deployments/$(panel_safe_name "$TARGET_ENV_DEPLOY")" "$patch_json" >/dev/null
              }

              restart_site_manager_nginx() {
                selector="app.kubernetes.io%2Finstance%3Dw7-sitemanager%2Capp.kubernetes.io%2Fname%3Dsite-manager-nginx"
                pods_json=$(k8s_get "/api/v1/namespaces/$NAMESPACE/pods?labelSelector=$selector")
                pod_names=$(printf '%s' "$pods_json" | jq -c '[.items[]? | select(.status.phase == "Running") | .metadata.name]')
                if [ "$(printf '%s' "$pod_names" | jq 'length')" = "0" ]; then
                  log_shell "skip nginx reload: reason=no-running-pod"
                  return 1
                fi
                exec_json=$(jq -n \
                  --arg namespace "$NAMESPACE" \
                  --arg containerName "site-manager-nginx" \
                  --argjson podNames "$pod_names" \
                  '{
                    namespace: $namespace,
                    podNames: $podNames,
                    containerName: $containerName,
                    command: ["nginx", "-s", "reload"],
                    tty: false
                  }')
                panel_post "/panel-api/v1/exec-all" "$exec_json" >/dev/null
                log_shell "nginx reload triggered: pods=$pod_names command=nginx -s reload"
              }

              run_shells_by_type() {
                start_params_json="$1"
                shell_type="$2"
                shell_items="$3"
                if [ "$shell_items" = "[]" ]; then
                  log_shell "skip shell type: type=$shell_type reason=empty"
                  return 0
                fi
                log_shell "run shell type start: type=$shell_type"
                printf '%s' "$shell_items" | jq -c '.[]' | while read -r shell_line; do
                  title=$(printf '%s' "$shell_line" | jq -r '.title // ""')
                  image=$(printf '%s' "$shell_line" | jq -r '.image // ""')
                  body=$(printf '%s' "$shell_line" | jq -r '.shell // ""')
                  if [ -z "$body" ]; then
                    log_shell "skip empty shell: type=$shell_type title=$title"
                    continue
                  fi
                  create_shell_job "$shell_type" "$title" "$image" "$body" "$start_params_json"
                done
                log_shell "run shell type success: type=$shell_type"
              }

              state_json=$(load_state)
              TARGET_ENV_DEPLOY=$(printf '%s' "$state_json" | jq -r '.data.target_env_deploy')
              TARGET_ENV_IMAGE=$(printf '%s' "$state_json" | jq -r '.data.target_env_image // ""')
              TARGET_ENV_VOLUMES=$(printf '%s' "$state_json" | jq -r '.data.target_env_volumes // "[]"')
              TARGET_ENV_VOLUME_MOUNTS=$(printf '%s' "$state_json" | jq -r '.data.target_env_volume_mounts // "[]"')
              TARGET_ENV_AFFINITY=$(printf '%s' "$state_json" | jq -r '.data.target_env_affinity // "{}"')
              TARGET_ENV_RESOURCES=$(printf '%s' "$state_json" | jq -r '.data.target_env_resources // "{}"')
              TARGET_ENV_SECURITY_CONTEXT=$(printf '%s' "$state_json" | jq -r '.data.target_env_security_context // "{}"')
              TARGET_ENV_IMAGE_PULL_POLICY=$(printf '%s' "$state_json" | jq -r '.data.target_env_image_pull_policy // "IfNotPresent"')
              OPERATION=$(printf '%s' "$state_json" | jq -r '.data.operation')
              DOMAIN=$(printf '%s' "$state_json" | jq -r '.data.domain')

              start_params_json=$(decode_b64_json "$START_PARAMS_ENV_B64" "{}")
              user_shells_json=$(decode_b64_json "$SHELLS_B64" "[]")

              if [ "$OPERATION" = "install" ]; then
                run_shells_by_type "$start_params_json" "pre-install" "$(printf '%s' "$user_shells_json" | jq '[.[] | select(.type == "pre-install" or .type == "requireinstall")]')"
                run_restart_patch
                restart_site_manager_nginx
                run_shells_by_type "$start_params_json" "post-install" "$(printf '%s' "$user_shells_json" | jq '[.[] | select(.type == "post-install" or .type == "install")]')"
              else
                run_shells_by_type "$start_params_json" "pre-upgrade" "$(printf '%s' "$user_shells_json" | jq '[.[] | select(.type == "pre-upgrade")]')"
                run_restart_patch
                restart_site_manager_nginx
                run_shells_by_type "$start_params_json" "post-upgrade" "$(printf '%s' "$user_shells_json" | jq '[.[] | select(.type == "post-upgrade" or .type == "upgrade")]')"
              fi
              run_shells_by_type "$start_params_json" "custom" "$(printf '%s' "$user_shells_json" | jq '[.[] | select(.type == "custom")]')"
              k8s_delete "/api/v1/namespaces/$NAMESPACE/configmaps/$STATE_CONFIG" >/dev/null
