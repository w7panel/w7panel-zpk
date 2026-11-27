{{/* Resource startup values are optional application inputs, never env entries. */}}
{{- define "common.installResourceLimits" -}}
{{- $result := dict "containers" (dict) "bandwidth" "" -}}
{{- $config := .Values.resourceConfig | default dict -}}
{{- $parameters := get $config "parameters" | default dict -}}
{{- $allWeights := get $config "weights" | default dict -}}
{{- $containers := list -}}
{{- range .Values.containers -}}
  {{- if not .isInitContainer -}}{{- $containers = append $containers .name -}}{{- end -}}
{{- end -}}
{{- range $kind, $name := $parameters -}}
  {{- $raw := toString (get $.Values $name | default "") | trim -}}
  {{- if $raw -}}
    {{- if not (regexMatch "^[0-9]+(\\.[0-9]+)?$" $raw) -}}{{- fail (printf "%s 资源参数必须为正数" $kind) -}}{{- end -}}
    {{- $number := float64 $raw -}}
    {{- if or (le $number 0.0) (gt $number 1000000000.0) -}}{{- fail (printf "%s 资源参数超出范围" $kind) -}}{{- end -}}
    {{- if eq $kind "bandwidth" -}}
      {{- $_ := set $result "bandwidth" (printf "%sM" $raw) -}}
    {{- else -}}
      {{- if eq (len $containers) 0 -}}{{- fail "资源配置需要普通应用容器" -}}{{- end -}}
      {{- $total := $number -}}
      {{- if eq $kind "cpu" -}}{{- $total = mulf $number 1000 | float64 -}}{{- end -}}
      {{- if eq $kind "memory" -}}{{- $total = mulf $number 1048576 | float64 -}}{{- end -}}
      {{- if or (lt $total 1.0) (ne $total (floor $total)) -}}{{- fail (printf "%s 资源参数精度无效" $kind) -}}{{- end -}}
      {{- $weights := deepCopy (get $allWeights $kind | default dict) -}}
      {{- $sum := 0.0 -}}
      {{- range $containers -}}{{- $sum = addf $sum (get $weights . | default 0) | float64 -}}{{- end -}}
      {{- if eq $sum 0.0 -}}
        {{- range $index, $container := $containers -}}
          {{- $weight := 1 -}}
          {{- if and (eq $kind "gpu") (gt $index 0) -}}{{- $weight = 0 -}}{{- end -}}
          {{- $_ := set $weights $container $weight -}}
          {{- $sum = addf $sum $weight | float64 -}}
        {{- end -}}
      {{- end -}}
      {{- $allocated := dict -}}
      {{- $remaining := int64 $total -}}
      {{- $first := "" -}}
      {{- range $containers -}}
        {{- $weight := get $weights . | default 0 | float64 -}}
        {{- if gt $weight 0.0 -}}
          {{- if not $first -}}{{- $first = . -}}{{- end -}}
          {{- $amount := mulf $total (divf $weight $sum) | floor | int64 -}}
          {{- $_ := set $allocated . $amount -}}
          {{- $remaining = sub $remaining $amount -}}
        {{- end -}}
      {{- end -}}
      {{- $_ := set $allocated $first (add (get $allocated $first) $remaining) -}}
      {{- range $container, $amount := $allocated -}}
        {{- if and (ne $kind "gpu") (lt (int64 $amount) 1) -}}{{- fail (printf "%s 资源限制过小，无法分配给全部容器" $kind) -}}{{- end -}}
        {{- $entry := get $result.containers $container | default (dict "limits" (dict)) -}}
        {{- $key := $kind -}}
        {{- $value := printf "%d" (int64 $amount) -}}
        {{- if eq $kind "cpu" -}}{{- $value = printf "%sm" $value -}}{{- end -}}
        {{- if eq $kind "gpu" -}}{{- $key = "nvidia.com/gpu" -}}{{- $_ := set $entry "requests" (dict $key $value) -}}{{- end -}}
        {{- $_ := set $entry.limits $key $value -}}
        {{- $_ := set $result.containers $container $entry -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- toJson $result -}}
{{- end -}}
