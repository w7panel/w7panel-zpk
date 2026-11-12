{{- range $stage := .Values.stages }}
{{- $tasks := list }}
{{- range $.Values.tasks }}
{{- if and (index $.Values.selected .id) (eq (int .stage) (int $stage)) }}
{{- $tasks = append $tasks . }}
{{- end }}
{{- end }}
{{- if $tasks }}
---
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $.Release.Name }}-stage-{{ $stage }}
  labels:
    w7.cc/workflow-run: {{ $.Release.Name | quote }}
  annotations:
    helm.sh/hook: post-install,post-upgrade
    helm.sh/hook-weight: {{ $stage | quote }}
    helm.sh/hook-delete-policy: before-hook-creation
spec:
  backoffLimit: 0
  activeDeadlineSeconds: {{ $.Values.deadlineSeconds }}
  ttlSecondsAfterFinished: 86400
  template:
    metadata:
      labels:
        w7.cc/workflow-run: {{ $.Release.Name | quote }}
    spec:
      serviceAccountName: {{ $.Release.Name }}
      automountServiceAccountToken: true
      restartPolicy: Never
      containers:
{{- range $tasks }}
        - name: {{ .id }}
          image: {{ if eq .action "build" }}{{ $.Values.buildImage | quote }}{{ else }}{{ $.Values.deployImage | quote }}{{ end }}
          command: [{{ if eq .action "build" }}"/busybox/sh", "/config/build.sh"{{ else }}"/bin/sh", "/config/deploy.sh"{{ end }}]
          env:
            - name: TASK_ID
              value: {{ .id | quote }}
            - name: SOURCE_TYPE
              value: {{ .source | quote }}
            - name: SOURCE_TASK
              value: {{ .source_task | quote }}
            - name: ORIGINAL_IMAGE
              value: {{ $.Values.run.original | quote }}
            - name: IMAGE_REPOSITORY
              value: {{ $.Values.run.repository | quote }}
            - name: REGISTRY_HOST
              value: {{ $.Values.run.host | quote }}
            - name: PLAIN_HTTP
              value: {{ $.Values.run.plainHTTP | quote }}
            - name: TARGET_IMAGE
              value: {{ printf "%s:%s%s" $.Values.run.repository $.Values.run.tag .tag_suffix | quote }}
            - name: RUN_NAMESPACE
              value: {{ $.Release.Namespace | quote }}
            - name: RESULTS_NAME
              value: {{ printf "%s-results" $.Release.Name | quote }}
            - name: TARGET_NAMESPACE
              value: {{ .k8s_namespace | quote }}
            - name: TARGET_KIND
              value: {{ .k8s_controller_type | quote }}
            - name: TARGET_NAME
              value: {{ .k8s_app_name | quote }}
            - name: TARGET_CONTAINERS
              value: {{ .k8s_container_name | quote }}
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: "2"
              memory: 2Gi
          volumeMounts:
            - name: scripts
              mountPath: /config
              readOnly: true
{{- if eq .action "build" }}
            - name: registry-auth-{{ .id }}
              mountPath: /kaniko/.docker
              readOnly: true
{{- else if index $.Values.remote .id }}
            - name: target-{{ .id }}
              mountPath: /target
              readOnly: true
{{- end }}
{{- end }}
      volumes:
        - name: scripts
          configMap:
            name: {{ $.Release.Name }}-scripts
{{- range $tasks }}
{{- if eq .action "build" }}
        - name: registry-auth-{{ .id }}
          secret:
            secretName: {{ $.Release.Name }}-auth-{{ .id }}
{{- end }}
{{- if index $.Values.remote .id }}
        - name: target-{{ .id }}
          secret:
            secretName: {{ $.Release.Name }}-{{ .id }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
