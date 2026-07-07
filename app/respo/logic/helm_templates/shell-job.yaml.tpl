{{- if gt (len .Values.jobs) 0 }}
  {{- $root := . }}
  {{- range $job := .Values.jobs }}
---
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "common.fullname" $root }}-job-{{ $job.name }}
  labels:
    {{- include "common.labels" $root | nindent 4 }}
    group: {{ $root.Release.Name }}
    w7.cc/group-name: {{ $root.Release.Name }}
    w7.cc/job-source: appgroup
  annotations:
  {{- if ne $job.type "custom" }}
    helm.sh/hook: {{ $job.type }}
    helm.sh/hook-weight: "{{ $job.weight }}"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
  {{- else }}
    w7.cc/custom-hook: 'true'
    {{- end }}
spec:
  backoffLimit: 2
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        group: {{ $root.Release.Name }}
        w7.cc/group-name: {{ $root.Release.Name }}
        w7.cc/job-source: appgroup
      annotations:
      {{- if $root.Values.podAnnotations }}
        {{- toYaml $root.Values.podAnnotations | nindent 8 }}
      {{- end }}
        {{- if $root.Values.annotations }}
        {{- toYaml $root.Values.annotations | nindent 12 }}
        {{- end }}
    spec:
      restartPolicy: Never
      serviceAccountName: {{ include "common.serviceAccountName" $root }}
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: w7.cc/identifie
                    operator: In
                    values:
                      - {{ $root.Values.app.identify | quote }}
              topologyKey: kubernetes.io/hostname
      {{- if $root.Values.volumes }}
      volumes:
        {{- include "common.volumesToYaml" (dict "root" $root "volumes" $root.Values.volumes) | nindent 8 }}
      {{- end }}
      containers:
        - name: {{ $job.name }}
          {{- if $job.image }}
          image: {{ $job.image | quote }}
          {{- else }}
          image: "{{ $job.container.image.repository }}:{{ $job.container.image.tag }}"
          {{- end }}
          imagePullPolicy: {{ $job.container.image.pullPolicy | default "IfNotPresent" }}
          command: ["/bin/sh", "-c"]
          args:
            - {{ $job.shell | quote }}
          env:
            {{- with $job.container.env }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
            {{- if $root.Values.startParams }}
            {{- range $k, $v := $root.Values.startParams }}
            - name: {{ $k | quote }}
              value: {{ tpl $v $root | quote }}
            {{- end }}
            {{- end }}
          {{- with $job.container.resources }}
          resources: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- $renderedVolumeMounts := include "common.jobVolumeMountsToYaml" (dict "root" $root "mounts" $job.container.volumeMounts) }}
          {{- if $renderedVolumeMounts }}
          volumeMounts: {{- $renderedVolumeMounts | nindent 12 }}
          {{- end }}
          {{- with $job.container.securityContext }}
          securityContext: {{- toYaml . | nindent 12 }}
          {{- end }}
    {{- end }}
{{- end }}
