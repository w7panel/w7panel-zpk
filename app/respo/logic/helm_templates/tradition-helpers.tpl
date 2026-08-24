{{/*
Resolve the concrete environment AppGroup selected for this traditional app.
The artifact fixes environmentIdentifie; the installer only supplies releaseName.
*/}}
{{- define "tradition.environmentAppGroup" -}}
{{- $releaseNameParam := printf "%s_RELEASE_NAME" (upper (replace "-" "_" .Values.tradition.environmentIdentifie)) -}}
{{- $releaseName := index .Values $releaseNameParam -}}
{{- $releaseName = required "传统应用必须指定环境应用 releaseName" $releaseName -}}
{{- $group := lookup "w7panel.w7.com/v1alpha1" "AppGroup" .Release.Namespace $releaseName -}}
{{- if empty $group -}}
{{- fail (printf "环境应用 AppGroup %s 不存在" $releaseName) -}}
{{- end -}}
{{- $annotations := default (dict) $group.metadata.annotations -}}
{{- if eq (default "" (index $annotations "w7.cc/parent-root")) "true" -}}
{{- fail (printf "AppGroup %s 是环境归类节点，不能作为传统应用安装目标" $releaseName) -}}
{{- end -}}
{{- $expectedIdentify := replace "_" "-" .Values.tradition.environmentIdentifie -}}
{{- $actualIdentify := replace "_" "-" (default "" (index $annotations "w7.cc/identifie")) -}}
{{- if ne $actualIdentify $expectedIdentify -}}
{{- fail (printf "环境应用类型不匹配，期望 %s，实际 %s" $expectedIdentify $actualIdentify) -}}
{{- end -}}
{{- $deployStatus := dig "status" "deployStatus" "" $group -}}
{{- if ne $deployStatus "deployed" -}}
{{- fail (printf "环境应用 %s 尚未部署完成" $releaseName) -}}
{{- end -}}
{{- toJson $group -}}
{{- end -}}

{{/* Return the selected environment domain as a filesystem-safe directory name. */}}
{{- define "tradition.codeInstallDirectory" -}}
{{- $group := include "tradition.environmentAppGroup" . | fromJson -}}
{{- $annotations := default (dict) $group.metadata.annotations -}}
{{- $releaseNameParam := printf "%s_RELEASE_NAME" (upper (replace "-" "_" .Values.tradition.environmentIdentifie)) -}}
{{- $releaseName := index .Values $releaseNameParam -}}
{{- $domain := default "" (index $annotations "w7.cc/default-domain") -}}
{{- $domain = required (printf "环境应用 %s 未配置默认域名" $releaseName) $domain -}}
{{- $domain = regexReplaceAll "^https?://" $domain "" -}}
{{- $domain = trimSuffix "/" $domain -}}
{{- if not (regexMatch "^[A-Za-z0-9._-]+$" $domain) -}}
{{- fail (printf "环境应用 %s 的默认域名无效: %s" $releaseName $domain) -}}
{{- end -}}
{{- $domain -}}
{{- end -}}
