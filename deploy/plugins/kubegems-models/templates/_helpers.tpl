{{/*
Return the proper image name
{{ include "common.images.image" ( dict "imageRoot" .Values.path.to.the.image "global" $) }}
*/}}
{{- define "kubegems.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if or (not $tag) (eq $tag "latest") -}}
    {{- $tag = printf "v%s" .root.Chart.AppVersion | toString -}}
{{- end -}}
{{- if .global.kubegemsVersion }}
    {{- $tag = .global.kubegemsVersion | toString -}}
{{- end }}
{{- if .global.imageRegistry }}
    {{- $registryName = .global.imageRegistry -}}
{{- end -}}
{{- if $registryName }}
    {{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
    {{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}

{{- define "kubegems.controller.fullname" -}}
{{- printf "%s-controller" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kubegems.controller.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (printf "%s" (include "kubegems.controller.fullname" .)) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{- define "kubegems.controller.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.controller.image "global" .Values.global "root" .) }}
{{- end -}}

{{- define "kubegems.store.fullname" -}}
{{- printf "%s-store" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kubegems.store.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.store.image "global" .Values.global "root" .) }}
{{- end -}}

{{- define "kubegems.sync.fullname" -}}
{{- printf "%s-sync" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kubegems.sync.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.sync.image "global" .Values.global) }}
{{- end -}}

{{- define "kubegems.sync.address" -}}
{{ printf "http://%s:%.0f" (include "kubegems.sync.fullname" .) (.Values.sync.service.ports.http) }}
{{- end -}}

{{- define "kubegems.mongodb.name" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "mongodb" "chartValues" .Values.mongodb "context" $) -}}
{{- end -}}

{{- define "kubegems.mongo.address" -}}
{{ printf "%s:%d" (include "kubegems.mongodb.name" .) 27017 }}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "kubegems.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.controller.image .Values.store.image) "global" .Values.global) -}}
{{- end -}}

{{/*
Return true if cert-manager required annotations for TLS signed certificates are set in the Ingress annotations
Ref: https://cert-manager.io/docs/usage/ingress/#supported-annotations
*/}}
{{- define "kubegems.ingress.certManagerRequest" -}}
{{ if or (hasKey . "cert-manager.io/cluster-issuer") (hasKey . "cert-manager.io/issuer") }}
    {{- true -}}
{{- end -}}
{{- end -}}

{{/*
Compile all warnings into a single message.
*/}}
{{- define "kubegems.models.validateValues" -}}
{{- $messages := list -}}
{{- $messages := without $messages "" -}}
{{- $message := join "\n" $messages -}}

{{- if $message -}}
{{-   printf "\nVALUES VALIDATION:\n%s" $message -}}
{{- end -}}
{{- end -}}

