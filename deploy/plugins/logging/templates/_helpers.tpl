{{- define "loki.address" -}}
{{- .Values.loki.address | default (printf "http://loki.%s:3100" .Release.Namespace) -}}
{{- end -}} 