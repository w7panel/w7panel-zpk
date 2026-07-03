{{- if or (hasKey .Values "containers") (gt (len .Values.containers) 0) }}
  {{- $hasJobs := false }}
  {{- range .Values.containers }}
    {{- if and (hasKey . "buildImageJobs") (gt (len .buildImageJobs) 0) }}
      {{- $hasJobs = true }}
    {{- end }}
  {{- end }}
  {{- if $hasJobs }}
    {{- $root := . }}
    {{/* 渲染所有 Job */}}
    {{- range $container := .Values.containers }}
      {{- range $job := $container.buildImageJobs }}
---
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "common.fullname" $root }}-{{ $container.name }}-job-build-image
  labels:
    {{- include "common.labels" $root | nindent 4 }}
    group: {{ $root.Release.Name }}
    w7.cc/job-source: appgroup
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 3
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        group: {{ $root.Release.Name }}
        w7.cc/job-source: appgroup
      {{- if $root.Values.podAnnotations }}
      annotations:
        {{- toYaml $root.Values.podAnnotations | nindent 8 }}
      {{- end }}
    spec:
      restartPolicy: Never
      volumes:
        -
          name: my-host
          hostPath:
            path: /
            type: ''
      containers:
         -
            name: docker-build
            image: ccr.ccs.tencentyun.com/afan-public/kaniko:w7console-new5-19
            command:
               - /kaniko/start.sh
            workingDir: /workspace
            env:
              -
                name: NOTIFY_COMPLETION_URL
                value: /
              -
                name: KANIKO_REGISTRY_MAP
                value: "index.docker.io=mirror.ccs.tencentyun.com;index.docker.io=registry.cn-hangzhou.aliyuncs.com;\n\tindex.docker.io=docker.m.daocloud.io;index.docker.io=docker.1panel.live"
              -
                name: EMBED
                value: 'true'
              -
                name: USER_AGENT
                value: release
              -
                name: MODULE_NAME
                value: {{ $job.identifie }}
              -
                name: DOCKER_AUTH
                value: >-
                  {"auths":{"registry.local.w7.cc":{"auth":"YWRtaW46dzctc2VjcmV0"}}}
              -
                name: DOWNLOAD_URL
                value: {{ $job.zipUrl }}
              -
                name: NOTIFY_FAILED_URL
                value: /
              -
                name: CURL_CA_BUNDLE
                value: /kaniko/ssl/certs/ca-certificates.crt
              -
                name: CONTEXT
                value: /workspace
              -
                name: DOCKER_FILE
                value: /workspace/{{ $job.dockerFilePath }}
              -
                name: ATTACHMENT_TYPE
                value: zip
              -
                name: PUSH_IMAGE
                value: {{ $job.buildImageName }}
              -
                name: INSECURE
                value: '--insecure --insecure-pull'
            resources: {}
            volumeMounts:
              -
                name: my-host
                mountPath: /host
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
