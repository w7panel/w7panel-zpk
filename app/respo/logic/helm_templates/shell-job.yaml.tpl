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
      {{- $podAnnotations := include "w7panel.podAnnotations" $root }}
      {{- if $podAnnotations }}
      annotations:
        {{- $podAnnotations | nindent 8 }}
      {{- end }}
    spec:
      {{- $jobSidecarVolumes := include "w7panel.sidecars.jobVolumes" $root }}
      {{- $jobSidecarInitContainers := include "w7panel.sidecars.jobInitContainers" $root }}
      {{- $jobSidecarHostAliases := include "w7panel.sidecars.jobHostAliases" $root }}
      restartPolicy: Never
      {{- if $jobSidecarHostAliases }}
      hostAliases:
        {{- $jobSidecarHostAliases | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "common.serviceAccountName" $root }}
      {{- with $root.Values.jobAffinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- if or $root.Values.volumes $jobSidecarVolumes }}
      volumes:
        {{- if $root.Values.volumes }}
        {{- include "common.volumesToYaml" (dict "root" $root "volumes" $root.Values.volumes) | nindent 8 }}
        {{- end }}
        {{- $jobSidecarVolumes | nindent 8 }}
      {{- end }}
      {{- if $jobSidecarInitContainers }}
      initContainers:
        {{- $jobSidecarInitContainers | nindent 8 }}
      {{- end }}
      containers:
        - name: {{ $job.name }}
          {{- if $job.image }}
          image: {{ $job.image | quote }}
          {{- else }}
          image: "{{ $job.container.image.repository }}:{{ tpl $job.container.image.tag $root }}"
          {{- end }}
          imagePullPolicy: {{ $job.container.image.pullPolicy | default "IfNotPresent" }}
          command: ["/bin/sh", "-c"]
          args:
            - {{ tpl $job.shell $root | quote }}
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
