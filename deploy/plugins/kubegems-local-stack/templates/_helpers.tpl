{{- define "observability.jaeger.address" -}}
{{- if .Values.observability.enabled  }}
    {{- printf "http://jaeger-operator-jaeger-query.%s:16686" .Values.observability.namespace }}
{{- else if .Values.observability.values.externalJaeger -}}
    {{- .Values.observability.values.externalJaeger.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.prometheus.address" -}}
{{- if .Values.monitoring.enabled  }}
    {{- printf "http://kube-prometheus-stack-prometheus.%s:9090" .Values.monitoring.namespace }}
{{- else if and .Values.monitoring.values .Values.monitoring.values.externalPrometheus -}}
    {{- .Values.monitoring.values.externalPrometheus.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.alertmanager.address" -}}
{{- if .Values.monitoring.enabled  }}
    {{- printf "http://kube-prometheus-stack-alertmanager.%s:9093" .Values.monitoring.namespace }}
{{- else if and .Values.monitoring.values .Values.monitoring.values.externalAlertmanager -}}
    {{- .Values.monitoring.values.externalAlertmanager.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.grafana.address" -}}
{{- if .Values.monitoring.enabled  }}
    {{- printf "http://kube-prometheus-stack-grafana.%s:80" .Values.monitoring.namespace }}
{{- else if and .Values.monitoring.values .Values.monitoring.values.externalGrafana -}}
    {{- .Values.monitoring.values.externalGrafana.address }}
{{- end -}}
{{- end -}}

{{- define "logging.loki.address" -}}
{{- if .Values.logging.enabled  }}
    {{- printf "http://loki.%s:3100" .Values.logging.namespace }}
{{- else if .Values.logging.values.externalLoki -}}
    {{- .Values.logging.values.externalLoki.address }}
{{- end -}}
{{- end -}}

{{- define "kubegems.local.agent.alert" -}}
    {{- printf "https://kubegems-local-agent.%s:8041/alert" (index .Values "kubegems-local" "namespace") }}
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
{{ include "common.container.runtime" . }}
*/}}
{{- define "common.container.runtime" -}}
{{- $globalRuntime := .Values.global.runtime -}}
{{- if $globalRuntime -}}
runtime: {{ $globalRuntime }}
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