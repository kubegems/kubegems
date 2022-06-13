{{- define "prometheus.address" -}}
{{- .Values.prometheus.address | default (printf "http://kube-prometheus-stack-prometheus.%s:9090" .Release.Namespace) -}}
{{- end -}} 

{{- define "alertmanager.address" -}}
{{- .Values.alertmanager.address | default (printf "http://kube-prometheus-stack-alertmanager.%s:9093" .Release.Namespace) -}}
{{- end -}} 
