{{- define "jaeger.address" -}}
    {{- printf "http://jaeger-query.%s:16686" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.collector.address" -}}
    {{- printf "http://jaeger-operator-jaeger-collector.%s:16686" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.collector.otel.address" -}}
    {{- printf "http://jaeger-operator-jaeger-collector.%s:14250" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.zipkin.address" -}}
    {{- printf "jaeger-operator-jaeger-collector.%s:9411" .Release.Namespace -}}
{{- end -}}
