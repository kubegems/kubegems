{{- define "prometheus.address" -}}
{{- .Values.prometheus.address | default (printf "http://kube-prometheus-stack-prometheus.%s:9090" .Release.Namespace) -}}
{{- end -}} 

{{- define "prometheus.rwrite.address" -}}
{{- .Values.prometheus.address | default (printf "http://kube-prometheus-stack-prometheus.%s:9090/api/v1/write" .Release.Namespace) -}}
{{- end -}} 

{{- define "alertmanager.address" -}}
{{- .Values.alertmanager.address | default (printf "http://kube-prometheus-stack-alertmanager.%s:9093" .Release.Namespace) -}}
{{- end -}} 

{{- define "kubegems.alertproxy.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.alertproxy.image "global" .Values.global) }}
{{- end -}}

{{- /*
{{ include "kubegems.images.image" ( dict "imageRoot" .Values.path.to.the.image "global" $) }}
*/ -}}
{{- define "kubegems.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
        {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if $registryName }}
    {{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
    {{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}