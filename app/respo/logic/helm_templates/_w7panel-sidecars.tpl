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
