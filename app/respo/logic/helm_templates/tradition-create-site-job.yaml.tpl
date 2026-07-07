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
      {{- end }}
    spec:
      restartPolicy: Never
      containers:
        - name: create-site-job
          image: zpk.w7.cc/public/site-manager:v1.2.12
          command:
            - sh
            - -c
            - |-
              set -eu

              PANEL_URL="{{ .Values.global.panel.innerUrl }}"
              PANEL_TOKEN="{{ .Values.global.panel.panelRealToken }}"
              NAMESPACE="{{ .Release.Namespace }}"
              SITE_MANAGER_NAMESPACE="{{ default .Release.Namespace .Values.global.siteManagerNamespace }}"
              SITE_MANAGER_URL="http://w7-sitemanager-site-manager.$SITE_MANAGER_NAMESPACE.svc.cluster.local:8000"
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

              K8S_ENV_APP_NAME="$NEW_ENV_K8S_APP"
              CREATED_ENV_APP=""
              CREATED_INGRESS_NAME=""
              CREATE_SITE_SUCCESS="false"

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

              cleanup_created_resources() {
                if [ "$CREATE_SITE_SUCCESS" = "true" ]; then
                  return 0
                fi
                if [ -n "$CREATED_INGRESS_NAME" ]; then
                  k8s_delete "/apis/networking.k8s.io/v1/namespaces/$NAMESPACE/ingresses/$CREATED_INGRESS_NAME" >/dev/null 2>&1 || true
                fi
                if [ -n "$CREATED_ENV_APP" ]; then
                  k8s_delete "/apis/apps/v1/namespaces/$NAMESPACE/deployments/$(panel_safe_name "$CREATED_ENV_APP")" >/dev/null 2>&1 || true
                fi
              }
              trap cleanup_created_resources EXIT
              k8s_delete "/api/v1/namespaces/$NAMESPACE/configmaps/$STATE_CONFIG" >/dev/null 2>&1 || true

              sm_post_maybe() {
                path="$1"
                data="$2"
                curl -sS -X POST \
                  -H "Content-Type: application/json" \
                  -d "$data" \
                  "$SITE_MANAGER_URL$path" || true
              }

              query_deploy() {
                deploy_name="$1"
                k8s_get "/apis/apps/v1/namespaces/$NAMESPACE/deployments/$(panel_safe_name "$deploy_name")"
              }

              create_ingress_if_needed() {
                if [ "$OPERATION" != "install" ]; then
                  return 0
                fi
                ingress_name="ing-$(date +%s)-$RANDOM"
                ingress_name=$(printf '%s' "$ingress_name" | tr '[:upper:]_' '[:lower:]-' | cut -c1-63 | sed 's/-$//')
                ingress_json=$(jq -n \
                  --arg ingressName "$ingress_name" \
                  --arg domain "$DOMAIN" \
                  --arg enableSsl "$ENABLE_SSL" \
                  --arg namespace "$NAMESPACE" \
                  '{
                    apiVersion: "networking.k8s.io/v1",
                    kind: "Ingress",
                    metadata: {
                      name: $ingressName,
                      namespace: $namespace,
                      annotations: ({
                        "kubernetes.io/ingress.class": "higress",
                        "higress.io/resource-definer": "higress"
                      } + (if $enableSsl == "true" then {
                        "higress.io/ssl-redirect": "false",
                        "w7.cc/ssl-redirect": "false",
                        "cert-manager.io/cluster-issuer": "w7-letsencrypt-prod",
                        "cert-manager.io/renew-before": "30m"
                      } else {} end)),
                      labels: {
                        "higress.io/resource-definer": "higress",
                        "app": "w7-sitemanager-site-manager-nginx",
                        "group": "w7-sitemanager",
                        "w7.cc/group-name": "w7-sitemanager",
                        "w7.cc/group-names": "{{ .Release.Name }}"
                      }
                    },
                    spec: {
                      rules: [
                        {
                          host: $domain,
                          http: {
                            paths: [
                              {
                                path: "/",
                                pathType: "Prefix",
                                backend: {
                                  service: {
                                    name: "w7-sitemanager-site-manager-nginx",
                                    port: { number: 80 }
                                  }
                                }
                              }
                            ]
                          }
                        }
                      ]
                    }
                  }
                  | if $enableSsl == "true" then .spec.tls = [{hosts: [$domain], secretName: ($domain + "-tls-secret")}] else . end')
                k8s_post "/apis/networking.k8s.io/v1/namespaces/$NAMESPACE/ingresses" "$ingress_json" >/dev/null
                CREATED_INGRESS_NAME="$ingress_name"
              }

              create_environment_app() {
                source_deploy_json="$1"
                safe_new_env_app=$(panel_safe_name "$NEW_ENV_K8S_APP")
                if k8s_get "/apis/apps/v1/namespaces/$NAMESPACE/deployments/$safe_new_env_app" >/dev/null 2>&1; then
                  echo "environment deployment already exists, reuse: $safe_new_env_app"
                  K8S_ENV_APP_NAME="$NEW_ENV_K8S_APP"
                  CREATED_ENV_APP=""
                  return 0
                fi

                yaml_copy_rules=$(printf '%s' "$source_deploy_json" | jq -r '.spec.template.metadata.annotations["w7.cc/yaml_copy"] // ""')
                if [ -n "$yaml_copy_rules" ] && [ "$yaml_copy_rules" != "null" ]; then
                  site_manager_deploy_json=$(query_deploy "w7-sitemanager-site-manager")
                  source_deploy_json=$(printf '%s' "$source_deploy_json" | jq -c \
                    --argjson sourceDeploy "$site_manager_deploy_json" \
                    --argjson copyRules "$yaml_copy_rules" \
                    '
                    def path_parts($p):
                      $p
                      | split(".")
                      | map(
                          capture("^(?<name>[^\\[]+)(?:\\[(?<idx>[0-9]+)\\])?$") as $m
                          | if $m.idx == null then [$m.name] else [$m.name, ($m.idx | tonumber)] end
                        )
                      | add;

                    reduce ($copyRules // [])[] as $rule
                      (.;
                        setpath(
                          path_parts($rule.target);
                          ($sourceDeploy | getpath(path_parts($rule.source)))
                        )
                      )
                    ')
                fi
                new_deploy_json=$(printf '%s' "$source_deploy_json" | jq \
                  --arg newName "$NEW_ENV_K8S_APP" \
                  --arg version "$ENV_VERSION" \
                  --arg releaseName "{{ .Release.Name }}" \
                  --arg namespace "$NAMESPACE" \
                  '
                  del(.metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.managedFields, .metadata.ownerReferences, .status)
                  | .metadata.name = $newName
                  | .metadata.namespace = $namespace
                  | .metadata.generation = 0
                  | .metadata.annotations = (.metadata.annotations // {})
                  | .metadata.labels = (.metadata.labels // {})
                  | .metadata.annotations["w7.cc/create-svc"] = "true"
                  | .metadata.annotations["title"] = $newName
                  | .metadata.annotations["w7.cc/parent-group-name"] = $releaseName
                  | .metadata.labels["w7.cc/parent-group-name"] = $releaseName
                  | .metadata.labels["app"] = $newName
                  | .spec.selector.matchLabels = (.spec.selector.matchLabels // {})
                  | .spec.selector.matchLabels["app"] = $newName
                  | .spec.template.metadata.labels = (.spec.template.metadata.labels // {})
                  | .spec.template.metadata.labels["app"] = $newName
                  | .spec.template.spec.containers[0].name = $newName
                  | ( .spec.template.metadata.annotations["w7.cc/image_template"] // "" ) as $tpl
                  | if $tpl != "" then .spec.template.spec.containers[0].image = ($tpl | gsub("\\{version\\}"; $version)) else . end
                  | .spec.template.spec.containers[0].env = (
                      (.spec.template.spec.containers[0].env // [])
                      | map(if .name == "METADATA_NAME" then .value = $newName | del(.valueFrom) else . end)
                    )
                  | .spec.template.spec.affinity = {
                      podAffinity: {
                        requiredDuringSchedulingIgnoredDuringExecution: [
                          {
                            labelSelector: {
                              matchExpressions: [
                                {
                                  key: "w7.cc/identifie",
                                  operator: "In",
                                  values: ["w7-sitemanager"]
                                }
                              ]
                            },
                            topologyKey: "kubernetes.io/hostname"
                          }
                        ]
                      }
                    }')
                k8s_post "/apis/apps/v1/namespaces/$NAMESPACE/deployments" "$new_deploy_json" >/dev/null

                K8S_ENV_APP_NAME="$NEW_ENV_K8S_APP"
                CREATED_ENV_APP="$NEW_ENV_K8S_APP"
              }

              resolve_target_env() {
                source_deploy_json=$(query_deploy "$ENV_GROUP")
                create_environment_app "$source_deploy_json"
              }

              get_site_env_app_name() {
                info_payload=$(jq -n --arg domain "$DOMAIN" '{domain:$domain}')
                site_info=$(sm_post_maybe "/api/site/info" "$info_payload")
                printf '%s' "$site_info" | jq -r '.data.site_environment.app_name // empty' 2>/dev/null || true
              }

              get_nginx_vhost_template() {
                deploy_json=$(query_deploy "$K8S_ENV_APP_NAME")
                printf '%s' "$deploy_json" | jq -r '.spec.template.metadata.annotations["w7.cc/nginx_vhost_template"] // .metadata.annotations["w7.cc/nginx_vhost_template"] // ""'
              }

              save_state() {
                target_env_deploy=$(get_site_env_app_name)
                if [ -z "$target_env_deploy" ]; then
                  target_env_deploy="$K8S_ENV_APP_NAME"
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

              resolve_target_env
              create_ingress_if_needed
              NGINX_VHOST_TEMPLATE=$(get_nginx_vhost_template)

              /home/rangine create:site \
                --title="$ENV_TITLE" \
                --name="$ENV_GROUP" \
                --language="$ENV_LANGUAGE" \
                --version="$ENV_VERSION" \
                --domain="$DOMAIN" \
                --ssl="$ENABLE_SSL" \
                --code-download-url="$CODE_DOWNLOAD_URL" \
                --app_name="$APP_IDENTIFY" \
                --k8s-app-name="$SITE_K8S_APP" \
                --k8s-env-app-name="$K8S_ENV_APP_NAME" \
                --nginx-vhost-template="$NGINX_VHOST_TEMPLATE" \
                -f /home/config.yaml

              CREATE_SITE_SUCCESS="true"
              save_state
