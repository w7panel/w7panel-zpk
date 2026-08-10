apiVersion: apps/v1
kind: {{ .Values.workload.kind }}
metadata:
  name: {{ include "common.fullname" . }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
    w7.cc/group-name: {{ .Release.Name }}
  annotations:
    title: {{ .Values.app.title | quote }}
    w7.cc.app/title: {{ .Values.app.title | quote }}
    w7.cc/create-svc: 'false'
    w7.cc/group-name: {{ .Release.Name }}
spec:
  {{- if not .Values.workload.isDaemonSet }}
  replicas: {{ .Values.replicas }}
  {{- end }}
  {{- if and .Values.workload.isDeployment .Values.workload.updateStrategy }}
  strategy: {{- toYaml .Values.workload.updateStrategy | nindent 4 }}
  {{- end }}
  {{- if .Values.workload.isStatefulSet }}
  serviceName: {{ include "common.serviceName" . }}
  podManagementPolicy: OrderedReady
  {{- end }}
  {{- if and (or .Values.workload.isDaemonSet .Values.workload.isStatefulSet) .Values.workload.updateStrategy }}
  updateStrategy: {{- toYaml .Values.workload.updateStrategy | nindent 4 }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "common.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
      {{- if .Values.podAnnotations }}
        {{- toYaml .Values.podAnnotations | nindent 8 }}
      {{- end }}
        {{- if .Values.annotations }}
        {{- toYaml .Values.annotations | nindent 12 }}
        {{- end }}
      labels:
        {{- include "common.selectorLabels" . | nindent 8 }}
        w7.cc/identifie: {{ .Values.app.identify | quote }}
    spec:
      {{- $root := . }}
      {{- $rootCtx := $ }}
      {{- if .Values.gpu.enable }}
      runtimeClassName: {{ .Values.gpu.driver }}
      {{- end }}

      {{- $podVolumes := .Values.volumes }}
      {{- $hasSidecarVolumes := false }}
      {{- range $sidecar := .Values.w7panelSidecars }}
        {{- if $sidecar.volumesTemplate }}
          {{- $hasSidecarVolumes = true }}
        {{- end }}
      {{- end }}
      {{- if .Values.workload.isStatefulSet }}
      {{- $claimTemplateNames := dict }}
      {{- range .Values.volumeClaimTemplates }}
      {{- $_ := set $claimTemplateNames .metadata.name true }}
      {{- end }}
      {{- $filteredVolumes := list }}
      {{- range .Values.volumes }}
      {{- if not (and .persistentVolumeClaim (hasKey $claimTemplateNames .name)) }}
      {{- $filteredVolumes = append $filteredVolumes . }}
      {{- end }}
      {{- end }}
      {{- $podVolumes = $filteredVolumes }}
      {{- end }}
      {{- if or $podVolumes $hasSidecarVolumes }}
      volumes:
        {{- if $podVolumes }}
        {{- include "common.volumesToYaml" (dict "root" . "volumes" $podVolumes) | nindent 8 }}
        {{- end }}
        {{- range $sidecar := .Values.w7panelSidecars }}
        {{- if $sidecar.volumesTemplate }}
        {{- $sidecarContext := index $root.Subcharts $sidecar.chart }}
        {{- include $sidecar.volumesTemplate $sidecarContext | nindent 8 }}
        {{- end }}
        {{- end }}
      {{- end }}

      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "common.serviceAccountName" . }}
      {{- if .Values.sharedStorageAffinity.targetSelectorApp }}
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: w7.cc/identifie
                    operator: In
                    values:
                      - {{ .Values.sharedStorageAffinity.targetSelectorApp | quote }}
              topologyKey: kubernetes.io/hostname
      {{- end }}

      containers:
      {{- range .Values.containers }}
      {{- if not .isInitContainer }}
        - name: {{ .name }}
          image: "{{ .image.repository }}:{{ .image.tag }}"
          imagePullPolicy: {{ .image.pullPolicy }}
          {{- with .command }}
          command: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .args }}
          args: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .ports }}
          ports: {{- toYaml . | nindent 12 }}
          {{- end }}
          env: 
            {{- with .env }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
            {{- if $root.Values.startParams }}
            {{- range $qkey, $qvalue := $root.Values.startParams }}
            - name: {{ $qkey }}
              value: {{ tpl $qvalue $rootCtx | quote }}
            {{- end }}
            {{- end }}
          {{- with .resources }}
          resources: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .volumeMounts }}
          volumeMounts: {{- tpl (toYaml .) $rootCtx | nindent 12 }}
          {{- end }}
          {{- with .livenessProbe }}
          livenessProbe: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .startupProbe }}
          startupProbe: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .lifecycle }}
          lifecycle: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .securityContext }}
          securityContext: {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- end }}
      {{- end }}
      {{- range $sidecar := .Values.w7panelSidecars }}
      {{- if $sidecar.containerTemplate }}
      {{- $sidecarContext := index $root.Subcharts $sidecar.chart }}
      {{- include $sidecar.containerTemplate $sidecarContext | nindent 8 }}
      {{- end }}
      {{- end }}

      {{- $hasInit := false }}
      {{- range .Values.containers }}
        {{- if .isInitContainer }}
          {{- $hasInit = true }}
        {{- end }}
      {{- end }}
      {{- range $sidecar := .Values.w7panelSidecars }}
        {{- if $sidecar.initTemplate }}
          {{- $hasInit = true }}
        {{- end }}
      {{- end }}
      {{- if $hasInit }}
      initContainers:
        {{- range .Values.containers }}
        {{- if .isInitContainer }}
        - name: {{ .name }}
          image: "{{ .image.repository }}:{{ .image.tag }}"
          imagePullPolicy: {{ .image.pullPolicy }}
          {{- with .command }}
          command: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .args }}
          args: {{- toYaml . | nindent 12 }}
          {{- end }}
          env:
            {{- with .env }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
            {{- if $root.Values.startParams }}
            {{- range $qkey, $qvalue := $root.Values.startParams }}
            - name: {{ $qkey }}
              value: {{ tpl $qvalue $rootCtx | quote }}
          {{- end }}
          {{- end }}
          {{- with .resources }}
          resources: {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .volumeMounts }}
          volumeMounts: {{- tpl (toYaml .) $rootCtx | nindent 12 }}
          {{- end }}
          {{- with .securityContext }}
          securityContext: {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- end }}
      {{- end }}
        {{- range $sidecar := .Values.w7panelSidecars }}
        {{- if $sidecar.initTemplate }}
        {{- $sidecarContext := index $root.Subcharts $sidecar.chart }}
        {{- include $sidecar.initTemplate $sidecarContext | nindent 8 }}
        {{- end }}
        {{- end }}
  {{- end }}
  {{- if and .Values.workload.isStatefulSet .Values.volumeClaimTemplates }}
  volumeClaimTemplates:
    {{- tpl (toYaml .Values.volumeClaimTemplates) . | nindent 4 }}
  {{- end }}
