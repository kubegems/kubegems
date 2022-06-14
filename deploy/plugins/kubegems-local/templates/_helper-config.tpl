{{- define "tracing.jaeger.address" -}}
{{- if and .Values.tracing .Values.tracing.jaeger  -}}
{{- .Values.tracing.jaeger.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.prometheus.address" -}}
{{- if and .Values.monitoring .Values.monitoring.prometheus -}}
{{- index .Values.monitoring.prometheus.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.alertmanager.address" -}}
{{- if and .Values.monitoring .Values.monitoring.alertmanager -}}
{{- .Values.monitoring.alertmanager.address }}
{{- end -}}
{{- end -}}

{{- define "monitoring.grafana.address" -}}
{{- if and .Values.monitoring .Values.monitoring.grafana -}}
{{- .Values.monitoring.grafana.address }}
{{- end -}}
{{- end -}}

{{- define "logging.loki.address" -}}
{{- if and .Values.logging .Values.logging.loki -}}
{{- .Values.logging.loki.address }}
{{- end -}}
{{- end -}}

{{- define "alert.address" -}}
{{- printf "https://%s.%s:%.f/alert" (include "kubegems-local.agent.fullname" .) .Release.Namespace .Values.agent.service.ports.http }}
{{- end -}}