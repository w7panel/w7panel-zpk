{{- $plugin := .Values.gatewayPlugin -}}
{{- $runtime := $plugin.runtime -}}
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: {{ .Release.Name }}
  namespace: higress-system
  annotations:
    higress.io/wasm-plugin-title: {{ .Values.app.title | quote }}
    higress.io/wasm-plugin-description: __APPLICATION_DESCRIPTION__
    w7.cc/plugin-support-global: {{ $plugin.supportGlobal | quote }}
    w7.cc/plugin-support-rule: {{ $plugin.supportRule | quote }}
  labels:
    w7.cc/group-name: {{ .Release.Name }}
    higress.io/wasm-plugin-name: "__APPLICATION_IDENTIFY__"
    higress.io/wasm-plugin-version: "__APPLICATION_VERSION__"
spec:
  url: {{ $runtime.url | quote }}
  phase: {{ $runtime.phase | quote }}
  priority: {{ $runtime.priority }}
  failStrategy: FAIL_OPEN
  defaultConfigDisable: {{ or (not $plugin.supportGlobal) (not $plugin.defaultEnabled) }}
  defaultConfig:
    {{ toYaml $plugin.defaultConfig | nindent 4 }}
  matchRules: []
