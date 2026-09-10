{{/* Return the PVC name injected from the selected environment dependency. */}}
{{- define "tradition.environmentStorageClaimName" -}}
{{- required "传统应用必须指定环境应用 PVC_NAME" .Values.PVC_NAME -}}
{{- end -}}

{{/* Return the selected environment domain as a filesystem-safe directory name. */}}
{{- define "tradition.codeInstallDirectory" -}}
{{- $domain := required "传统应用必须指定环境应用 DOMAIN_URL" .Values.DOMAIN_URL -}}
{{- $domain = regexReplaceAll "^https?://" $domain "" -}}
{{- $domain = trimSuffix "/" $domain -}}
{{- if not (regexMatch "^[A-Za-z0-9._-]+$" $domain) -}}
{{- fail (printf "环境应用域名无效: %s" $domain) -}}
{{- end -}}
{{- $domain -}}
{{- end -}}
