{{/* Return the PVC name injected from the selected environment dependency. */}}
{{- define "plugin.traditionStorageClaimName" -}}
{{- required "应用插件必须指定传统应用 PVC_NAME" .Values.PVC_NAME -}}
{{- end -}}

{{/* Use the target application's release, never the plugin's own release. */}}
{{- define "plugin.codeInstallDirectory" -}}
{{- $identify := required "应用插件必须指定传统应用标识" .Values.plugin.traditionIdentifie | replace "_" "-" | lower -}}
{{- $param := printf "%s_RELEASE_NAME" ($identify | replace "-" "_" | replace "." "_" | upper) -}}
{{- $releaseName := required "应用插件必须指定传统应用 Release 名称" (index .Values $param) -}}
{{- if or (gt (len $releaseName) 63) (not (regexMatch "^[a-z0-9]([a-z0-9-]*[a-z0-9])?$" $releaseName)) -}}
  {{- fail "invalid traditional application release name" -}}
{{- end -}}
{{- $releaseName -}}
{{- end -}}
