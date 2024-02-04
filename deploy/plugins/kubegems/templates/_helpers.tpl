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
{{- if and .global.imageRegistry (or (eq $registryName "docker.io") (not $registryName))  -}}
    {{- $registryName = .global.imageRegistry -}}
{{- end -}}
{{- if $registryName }}
    {{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
    {{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}

{{- define "kubegems.fullname" -}}
{{- printf "%s" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kubegems.pvc.name" -}}
    {{- default (printf "%s-data" (include "kubegems.fullname" .)) .Values.persistence.existingClaim -}}
{{- end -}}

{{/*
Return the proper kubegems dashboard name
*/}}
{{- define "kubegems.dashboard.fullname" -}}
{{- printf "%s-dashboard" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the proper kubegems image name
*/}}
{{- define "kubegems.dashboard.image" -}}
{{- include "kubegems.images.image" (dict "imageRoot" .Values.dashboard.image "global" .Values.global "root" .) -}}
{{- end -}}

{{- define "kubegems.api.address" -}}
    {{- include "kubegems.api.fullname" . -}}:{{- .Values.api.service.ports.http -}}
{{- end -}}

{{- define "kubegems.msgbus.address" -}}
    {{- include "kubegems.msgbus.fullname" . -}}:{{- .Values.api.service.ports.http -}}
{{- end -}}

{{- define "kubegems.pai.address" -}}
{{- if index .Values "kubegems-pai" "enabled"  -}}
kubegems-pai-api.kubegems-pai
{{- else -}}
{{- include "kubegems.api.address" . -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems.init.fullname" -}}
{{- if .Values.global.kubegemsVersion -}}
{{- (printf "%s-init-%s" (include "common.names.fullname" .) .Values.global.kubegemsVersion) | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-init" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper kubegems.api image name
*/}}
{{- define "kubegems.init.image" -}}
{{- include "kubegems.images.image" (dict "imageRoot" .Values.init.image "global" .Values.global "root" .) -}}
{{- end -}}

{{- define "kubegems.init.charts.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.init.charts.image "global" .Values.global "root" .) }}
{{- end -}}

{{- define "kubegems.init.charts.fullname" -}}
{{- if .Values.global.kubegemsVersion -}}
    {{- (printf "%s-charts-init-%s" (include "common.names.fullname" .) .Values.global.kubegemsVersion ) | trunc 63 | trimSuffix "-" -}}
{{- else -}}
    {{- printf "%s-charts-init" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems.api.fullname" -}}
{{- printf "%s-api" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the api jwt secretName
*/}}
{{- define "kubegems.api.jwt.secretName" -}}
{{- if .Values.api.jwt.secretName -}}
    {{- .Values.api.jwt.secretName -}}
{{- else }}
    {{- printf "%s-jwt" (include "kubegems.api.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper kubegems.api image name
*/}}
{{- define "kubegems.api.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.api.image "global" .Values.global "root" .) }}
{{- end -}}

{{/*
Return the proper kubegems msgbus name
*/}}
{{- define "kubegems.msgbus.fullname" -}}
{{- printf "%s-msgbus" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the proper kubegems image name
*/}}
{{- define "kubegems.msgbus.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.msgbus.image "global" .Values.global "root" .) }}
{{- end -}}

{{/*
Return the proper kubegems msgbus name
*/}}
{{- define "kubegems.worker.fullname" -}}
{{- printf "%s-worker" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the proper kubegems image name
*/}}
{{- define "kubegems.worker.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.worker.image "global" .Values.global "root" .) }}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "kubegems.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.api.image .Values.msgbus.image .Values.worker.image) "global" .Values.global) -}}
{{- end -}}
 
{{/*
Validate kubegems api configuration
*/}}
{{- define "kubegems.validateValues.api" -}}
{{- end -}}

{{/*
Compile all warnings into a single message.
*/}}
{{- define "kubegems.validateValues" -}}
{{- $messages := list -}}
{{- $messages := append $messages (include "kubegems.validateValues.api" .) -}}
{{- $messages := without $messages "" -}}
{{- $message := join "\n" $messages -}}

{{- if $message -}}
{{-   printf "\nVALUES VALIDATION:\n%s" $message -}}
{{- end -}}
{{- end -}}

