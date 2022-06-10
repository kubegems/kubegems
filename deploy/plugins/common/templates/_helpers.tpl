{{/*
    namespace:{{ include "common.component.namespace" "" }}
*/}}
{{- define "common.component.namespace" -}}
{{- "kubegems-local" -}}
{{- end -}}

{{/*
    name:{{ include "common.component.values" . }}
    name:{{ include "common.component.values" "logging" }}
*/}}
{{- define "common.component.name" -}}
{{ if typeIs "string" . }}
    {{- printf "kubegems-%v-values" . -}}
{{- else -}}
    {{- printf "kubegems-%v-values" .Release.Name -}}
{{- end -}}
{{- end -}}

{{- define "common.component.configmap" -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "common.component.name" . }}
  namespace: {{ include "common.component.namespace" . }}
{{- end -}}


{{/*
{{ $lokiaddress := include "common.component.values" "logging.loki.address" }}
*/}}
{{- define "common.component.values" -}}
{{- $splits := regexSplit "\\." . 2 -}}
{{- $name := index $splits 0 -}}
{{- $key := index $splits 1 -}}

{{- $cm := lookup "v1" "ConfigMap" (include "common.component.namespace" . ) (include "common.component.name" $name ) -}}
{{- if eq $key "enabled" -}}
    {{-  empty $cm -}}
{{- else if not (empty $cm) -}}
    {{- index $cm "data" $key -}}
{{- end -}}
{{- end -}}

{{/*
{{ include "common.images.registry" . }}
*/}}
{{- define "common.images.registry" -}}
{{- $globalRegistry := .Values.global.imageRegistry -}}
{{- if $globalRegistry -}}
registry: {{ $globalRegistry }}
{{- end -}}
{{- end -}}

{{/*
{{ include "common.images.repository" ( dict "default" "library/alpine" "context" .) }}
*/}}
{{- define "common.images.repository" -}}
{{- $repository := .repository -}}
{{- $registry := .registry -}}
{{- $key := .key -}}
{{- if not $key -}}
  {{- $key = "repository" -}}
{{- end -}}
{{- $globalRegistry := .context.Values.global.imageRegistry -}}
{{- $globalRepository := .context.Values.global.imageRepository -}}
{{- if and $registry $globalRegistry -}}
    {{- $registry = $globalRegistry -}}
{{- end -}}
{{- if $globalRepository -}}
    {{- $repository = printf "%s%s" $globalRepository (regexFind "/.*" $repository)  -}}
{{- end -}}
{{- if or $globalRegistry $globalRepository -}}
    {{- if $registry -}}
        {{- printf "%s: %s/%s" $key $registry $repository -}}
    {{- else -}}
        {{- printf "%s: %s" $key $repository -}}
    {{- end -}}
{{- end -}}
{{- end -}}