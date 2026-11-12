apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-results
data: {}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-scripts
data:
{{- range .Values.tasks }}
{{- if and (index $.Values.selected .id) (eq .action "build") }}
  {{ .id }}-dockerfile: {{ .dockerfile | quote }}
{{- end }}
{{- end }}
  build.sh: |
    #!/busybox/sh
    set -eu
    CURL=/kaniko/curl
    API="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${RUN_NAMESPACE}/configmaps/${RESULTS_NAME}"
    TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
    api() { "$CURL" --fail --silent --show-error --connect-timeout 10 --max-time 60 --cacert /var/run/secrets/kubernetes.io/serviceaccount/ca.crt -H "Authorization: Bearer $TOKEN" "$@" "$API"; }
    base="$ORIGINAL_IMAGE"
    if [ "$SOURCE_TYPE" = task ]; then
      document=$(api)
      base=$(printf '%s' "$document" | sed -n 's/.*"'"$SOURCE_TASK"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
      [ -n "$base" ] || { echo "来源任务没有构建结果" >&2; exit 1; }
    fi
    mkdir -p /workspace
    printf 'FROM %s\n' "$base" > /workspace/Dockerfile
    cat "/config/${TASK_ID}-dockerfile" >> /workspace/Dockerfile
    if [ "$PLAIN_HTTP" = true ]; then
      set -- "--insecure-registry=$REGISTRY_HOST"
    else
      set --
    fi
    /kaniko/executor --context=/workspace --dockerfile=/workspace/Dockerfile --destination="$TARGET_IMAGE" --digest-file=/workspace/digest --cache=false "$@"
    digest=$(cat /workspace/digest)
    case "$digest" in sha256:*) ;; *) echo "构建结果 digest 无效" >&2; exit 1;; esac
    result="${IMAGE_REPOSITORY}@${digest}"
    printf '%s' "$result" > /dev/termination-log
    body=$(printf '{"data":{"%s":"%s"}}' "$TASK_ID" "$result")
    api -X PATCH -H 'Content-Type: application/merge-patch+json' --data "$body" > /dev/null
  deploy.sh: |
    #!/bin/sh
    set -eu
    image="$ORIGINAL_IMAGE"
    if [ "$SOURCE_TYPE" = task ]; then
      image=$(kubectl -n "$RUN_NAMESPACE" get configmap "$RESULTS_NAME" -o "jsonpath={.data.${SOURCE_TASK}}")
      [ -n "$image" ] || { echo "来源任务没有构建结果" >&2; exit 1; }
    fi
    if [ -f /target/config ]; then export KUBECONFIG=/target/config; fi
    set --
    previous_ifs="$IFS"
    IFS=,
    for container in $TARGET_CONTAINERS; do set -- "$@" "$container=$image"; done
    IFS="$previous_ifs"
    kubectl --request-timeout=60s -n "$TARGET_NAMESPACE" set image "$TARGET_KIND/$TARGET_NAME" "$@"

