{{- define "observability.jaeger.address" -}}
{{- if .Values.observability.enabled  }}
    {{- printf "http://jaeger-query.%s:16686" .Values.observability.namespace }}
{{- else -}}
    {{- .Values.observability.jaeger.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.prometheus.address" -}}
{{- if .Values.monitoring.enabled  }}
    {{- printf "http://kube-prometheus-stack-prometheus.%s:9090" .Values.monitoring.namespace }}
{{- else -}}
    {{- .Values.monitoring.prometheus.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.alertmanager.address" -}}
{{- if .Values.monitoring.enabled  }}
    {{- printf "http://kube-prometheus-stack-alertmanager.%s:9093" .Values.monitoring.namespace }}
{{- else -}}
    {{- .Values.monitoring.alertmanager.address }}
{{- end -}}
{{- end -}}

{{- define "logging.loki.address" -}}
{{- if .Values.logging.enabled  }}
    {{- printf "http://loki-stack.%s:3100" .Values.logging.namespace }}
{{- else -}}
    {{- .Values.logging.loki.address }}
{{- end -}}
{{- end -}}

{{- define "kubegems.local.agent.alert" -}}
    {{- printf "https://kubegems-local-agent.%s:8041/alert" .Values.kubegems.local.namespace }}
{{- end -}}