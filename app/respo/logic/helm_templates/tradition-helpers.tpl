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

{{/*
Resolve the PVC used by the selected environment application. The PVC belongs
to the dependency release, so read the claim from that release's workload
instead of accepting a PVC_NAME from the traditional application or deriving
a name from the release. The environment packer reserves the site-storage
volume name for this shared claim, so other PVCs in the same workload are
ignored.
*/}}
{{- define "tradition.environmentStorageClaimName" -}}
{{- $releaseNameParam := printf "%s_RELEASE_NAME" (upper (replace "-" "_" .Values.tradition.environmentIdentifie)) -}}
{{- $releaseName := required "传统应用必须指定环境应用 releaseName" (index .Values $releaseNameParam) -}}
{{- $group := include "tradition.environmentAppGroup" . | fromJson -}}
{{- $claimName := "" -}}
{{- range $item := (default (list) (dig "status" "items" (list) $group)) -}}
  {{- $kind := default "" (index $item "kind") -}}
  {{- $name := default "" (index $item "name") -}}
  {{- $apiVersion := default "apps/v1" (index $item "apiVersion") -}}
  {{- if and (eq $claimName "") (ne $name "") (or (eq $kind "Deployment") (eq $kind "StatefulSet") (eq $kind "DaemonSet")) -}}
    {{- $resource := lookup $apiVersion $kind .Release.Namespace $name -}}
    {{- if $resource -}}
      {{- $spec := default (dict) (index $resource "spec") -}}
      {{- $template := default (dict) (index $spec "template") -}}
      {{- $podSpec := default (dict) (index $template "spec") -}}
      {{- range $volume := (default (list) (index $podSpec "volumes")) -}}
        {{- $claim := dig "persistentVolumeClaim" "claimName" "" $volume -}}
        {{- if and (eq (default "" (index $volume "name")) "site-storage") (eq $claimName "") (ne $claim "") -}}
          {{- $claimName = $claim -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if eq $claimName "" -}}
  {{- fail (printf "环境应用 %s 的工作负载未找到 site-storage PVC claimName" $releaseName) -}}
{{- end -}}
{{- $claimName -}}
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
