{{- define "jaeger.address" -}}
    {{- printf "http://jaeger-query.%s:16686" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.collector.address" -}}
    {{- printf "jaeger-operator-jaeger-collector.%s:14268" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.proto.address" -}}
    {{- printf "jaeger-operator-jaeger-collector.%s:14250" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.otlp.grpc.address" -}}
    {{- printf "jaeger-operator-jaeger-collector.%s:4317" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.otlp.http.address" -}}
    {{- printf "jaeger-operator-jaeger-collector.%s:4318" .Release.Namespace -}}
{{- end -}}

{{- define "jaeger.zipkin.address" -}}
    {{- printf "jaeger-operator-jaeger-collector.%s:9411" .Release.Namespace -}}
{{- end -}}
