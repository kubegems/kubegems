{{- define "jaeger.address" -}}
    {{- printf "http://jaeger-query.%s:16686" .Release.Namespace -}}
{{- end -}}

 