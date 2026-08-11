{{- define "w7panel.sidecars.podAnnotations" -}}
{{- $root := . -}}
{{- $annotations := dict -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- with $sidecar.podAnnotationsTemplate -}}
    {{- $context := index $root.Subcharts $sidecar.chart -}}
    {{- $rendered := include . $context | fromYaml | default dict -}}
    {{- range $key, $value := $rendered -}}
      {{- $_ := set $annotations $key $value -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if $annotations -}}
{{- toYaml $annotations -}}
{{- end -}}
{{- end -}}

{{- define "w7panel.podAnnotations" -}}
{{- $annotations := dict -}}
{{- range $key, $value := (.Values.podAnnotations | default dict) -}}
  {{- $_ := set $annotations $key $value -}}
{{- end -}}
{{- range $key, $value := (.Values.annotations | default dict) -}}
  {{- $_ := set $annotations $key $value -}}
{{- end -}}
{{- $sidecarAnnotations := include "w7panel.sidecars.podAnnotations" . | fromYaml | default dict -}}
{{- range $key, $value := $sidecarAnnotations -}}
  {{- $_ := set $annotations $key $value -}}
{{- end -}}
{{- if $annotations -}}
{{- toYaml $annotations -}}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.mergeHostAliases" -}}
{{- $root := .root -}}
{{- $jobOnly := .jobOnly -}}
{{- $aliasesByIP := dict -}}
{{- $hostnameIPs := dict -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- if or (not $jobOnly) $sidecar.jobContainerTemplate -}}
    {{- with $sidecar.hostAliasesTemplate -}}
      {{- $context := index $root.Subcharts $sidecar.chart -}}
      {{- range $item := (include . $context | fromYamlArray | default list) -}}
        {{- $ip := toString $item.ip -}}
        {{- if not $ip -}}
          {{- fail (printf "sidecar %s rendered a hostAliases item without ip" $sidecar.chart) -}}
        {{- end -}}
        {{- $existing := get $aliasesByIP $ip | default (dict "ip" $ip "hostnames" (list)) -}}
        {{- $hostnames := get $existing "hostnames" | default list -}}
        {{- range $hostname := ($item.hostnames | default list) -}}
          {{- $hostname = toString $hostname -}}
          {{- if hasKey $hostnameIPs $hostname -}}
            {{- $existingIP := get $hostnameIPs $hostname -}}
            {{- if ne $existingIP $ip -}}
              {{- fail (printf "sidecar hostAliases conflict for %s: %s and %s" $hostname $existingIP $ip) -}}
            {{- end -}}
          {{- else -}}
            {{- $_ := set $hostnameIPs $hostname $ip -}}
          {{- end -}}
          {{- if not (has $hostname $hostnames) -}}
            {{- $hostnames = append $hostnames $hostname -}}
          {{- end -}}
        {{- end -}}
        {{- $_ := set $existing "hostnames" $hostnames -}}
        {{- $_ := set $aliasesByIP $ip $existing -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- $items := list -}}
{{- range $ip := (keys $aliasesByIP | sortAlpha) -}}
  {{- $items = append $items (get $aliasesByIP $ip) -}}
{{- end -}}
{{- if $items -}}
{{ toYaml $items }}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.hostAliases" -}}
{{- include "w7panel.sidecars.mergeHostAliases" (dict "root" . "jobOnly" false) -}}
{{- end -}}

{{- define "w7panel.sidecars.jobHostAliases" -}}
{{- include "w7panel.sidecars.mergeHostAliases" (dict "root" . "jobOnly" true) -}}
{{- end -}}

{{- define "w7panel.sidecars.volumes" -}}
{{- $root := . -}}
{{- $items := list -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- with $sidecar.volumesTemplate -}}
    {{- $items = concat $items (include . (index $root.Subcharts $sidecar.chart) | fromYamlArray | default list) -}}
  {{- end -}}
{{- end -}}
{{- if $items -}}
{{ toYaml $items }}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.initContainers" -}}
{{- $root := . -}}
{{- $items := list -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- with $sidecar.initTemplate -}}
    {{- $items = concat $items (include . (index $root.Subcharts $sidecar.chart) | fromYamlArray | default list) -}}
  {{- end -}}
{{- end -}}
{{- if $items -}}
{{ toYaml $items }}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.containers" -}}
{{- $root := . -}}
{{- $items := list -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- with $sidecar.containerTemplate -}}
    {{- $items = concat $items (include . (index $root.Subcharts $sidecar.chart) | fromYamlArray | default list) -}}
  {{- end -}}
{{- end -}}
{{- if $items -}}
{{ toYaml $items }}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.jobInitContainers" -}}
{{- $root := . -}}
{{- $items := list -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- with $sidecar.jobContainerTemplate -}}
    {{- $context := index $root.Subcharts $sidecar.chart -}}
    {{- with $sidecar.initTemplate -}}
      {{- $items = concat $items (include . $context | fromYamlArray | default list) -}}
    {{- end -}}
    {{- $items = concat $items (include . $context | fromYamlArray | default list) -}}
  {{- end -}}
{{- end -}}
{{- if $items -}}
{{ toYaml $items }}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.jobVolumes" -}}
{{- $root := . -}}
{{- $items := list -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- if $sidecar.jobContainerTemplate -}}
    {{- with $sidecar.volumesTemplate -}}
      {{- $items = concat $items (include . (index $root.Subcharts $sidecar.chart) | fromYamlArray | default list) -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if $items -}}
{{ toYaml $items }}
{{- end -}}
{{- end -}}

{{- define "w7panel.sidecars.resources" -}}
{{- $root := . -}}
{{- range $sidecar := ($root.Values.w7panelSidecars | default list) -}}
  {{- with $sidecar.resourcesTemplate -}}
    {{- include . (index $root.Subcharts $sidecar.chart) -}}
  {{- end -}}
{{- end -}}
{{- end -}}
