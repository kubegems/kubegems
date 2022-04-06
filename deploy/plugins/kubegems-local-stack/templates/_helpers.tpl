{{/*
Return the proper kubegems dashboard name
*/}}
{{- define "observability.jaeger.address" -}}
{{- if .Values.observability.enabled  }}
    {{- printf "" .Values.observability.jaeger.address }}
{{- else -}}
    {{- .Values.observability.jaeger.address }}
{{- end -}}
{{- end -}}
